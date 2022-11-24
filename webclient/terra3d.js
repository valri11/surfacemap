import './terra3d.css';
import * as env from './env.json';
import * as places from './terra3dplaces.json';
import Procedural from 'procedural-gl';
import autoComplete from '@tarekraafat/autocomplete.js';

const placesList = document.getElementById('places-list');
const placesListOverlay = document.getElementById('places-list-overlay');
const title = document.getElementById('title');
//const subtitle = document.getElementById('subtitle');

//console.log(JSON.stringify(places.features))

function loadPlace(feat) {
    console.log(JSON.stringify(feat))

    const { name } = feat.properties;
    const [longitude, latitude] = feat.geometry.coordinates;
    Procedural.displayLocation( { latitude, longitude } );
    title.innerHTML = name;
    //subtitle.innerHTML = `${height}m`;
    placesListOverlay.classList.add( 'hidden' );
}

title.addEventListener( 'click', () => {
    placesListOverlay.classList.remove( 'hidden' );
} );

places.features.map((feat, i) => {
    console.log(JSON.stringify(feat))
    const li = document.createElement( 'li' );
    let p = document.createElement( 'p' );
    p.innerHTML = feat.properties.name;
    li.appendChild( p );
    p = document.createElement( 'p' );
    //p.innerHTML = i + 1;
    li.appendChild( p );
    placesList.appendChild( li );
    li.addEventListener( 'click', () => loadPlace(feat));
});

const basemap_default = 'arc';
var basemap = (new URLSearchParams(window.location.search)).get('basemap');

if (basemap != basemap_default) {
    if (!(basemap == 'arc' || basemap == 'osm' || basemap == 'nea')) {
        basemap = basemap_default;
        window.location.replace(`?basemap=${basemap}`);
    }
}

const container = document.getElementById( 'map' );
    
// Custom datasource definition
var datasource = {
  elevation: {
    apiKey: '',
    attribution: 'Golemresearch',
    pixelEncoding: 'terrarium',
    maxZoom: 12,
    urlFormat: `${env.elevation.urlTemplate}`
  },
  imagery: {
    attribution: `${env.arcgis.attribution}`,
    apiKey: `${env.arcgis.apikey}`,
    urlFormat: `${env.arcgis.urlTemplate}`
  }
}

if (basemap == 'arc') {
    datasource.imagery.attribution = `${env.arcgis.attribution}`;
    datasource.imagery.urlFormat = `${env.arcgis.urlTemplate}`;
} else if (basemap == 'osm') {
    datasource.imagery.attribution = `${env.osm.attribution}`;
    datasource.imagery.urlFormat = `${env.osm.urlTemplate}`;
} else if (basemap == 'nea') {
    datasource.imagery.attribution = `${env.nearmap.attribution}`;
    datasource.imagery.apiKey = `${env.nearmap.apikey}`;
    datasource.imagery.urlFormat = `${env.nearmap.urlTemplate}`;
}

Procedural.init( { container, datasource } );
Procedural.setCameraModeControlVisible( true );
Procedural.setCompassVisible( true );
Procedural.setUserLocationControlVisible( true );
Procedural.setRotationControlVisible( true );
Procedural.setZoomControlVisible( true );

if (places.features.length > 0) {
    loadPlace(places.features[0]);
}

navigator.geolocation.watchPosition(
  function (pos) {
    const coords = [pos.coords.longitude, pos.coords.latitude];
    //const accuracy = circular(coords, pos.coords.accuracy);
    Procedural.setUserLocation(pos);
    //title.innerHTML = `${coords[0]}, ${coords[1]}`;
  },
  function (error) {
    alert(`ERROR: ${error.message}`);
  },
  {
    enableHighAccuracy: true,
  }
);


const autoCompleteJS = new autoComplete({
    placeHolder: "(Lat,Lon)",
    data: {
    src: async (query) => {
      try {
            var arrCoord = query.split(",").map(Number);
            if (arrCoord.length != 2) {
                return [];
            }

            var arr = [query];
            return arr;
      } catch (error) {
            return error;
      }
    }
    },
    resultItem: {
        highlight: true
    },
    events: {
        input: {
            selection: (event) => {
                const selection = event.detail.selection.value;
                autoCompleteJS.input.value = selection;
                
                var feat = {
                    "type": "Feature",
                    "geometry": {"type":"Point", 
                        "coordinates":[]},
                        "properties":{"name":""}
                };
                var arrCoord = selection.split(",").map(Number);
                feat.geometry.coordinates = [arrCoord[1], arrCoord[0]];
                feat.properties['name'] = selection;
                loadPlace(feat);
            }
        }
    }
});
