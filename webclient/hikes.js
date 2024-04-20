import './hikes.css';
import * as env from './env.json';
import Map from 'ol/Map';
import View from 'ol/View';
import {Tile as TileLayer, VectorTile as VectorTileLayer, Image as ImageLayer} from 'ol/layer';
import {TileDebug, OSM, XYZ, VectorTile, Raster} from 'ol/source';
import {GeoJSON, GPX, MVT} from 'ol/format';
import {createStringXY} from 'ol/coordinate';
import {fromLonLat, getPointResolution, transform} from 'ol/proj';
import Overlay from 'ol/Overlay';
import {Fill, Stroke, Circle, Style, Text} from 'ol/style';
import {createXYZ} from 'ol/tilegrid';
import {Attribution, MousePosition, defaults as defaultControls} from 'ol/control';
import sync from 'ol-hashed';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import {circular} from 'ol/geom/Polygon';
import Control from 'ol/control/Control';
import autoComplete from '@tarekraafat/autocomplete.js';

import 'ol/ol.css';
import 'ol-ext/dist/ol-ext.css';
import './ctrls-over.css';

import LayerSwitcher from 'ol-ext/control/LayerSwitcher';
import Button from 'ol-ext/control/Button';
import Profile from 'ol-ext/control/Profile';
import Hover from 'ol-ext/interaction/Hover';

import * as places from './hikes_places.json';

const placesList = document.getElementById('places-list');
const placesListOverlay = document.getElementById('places-list-overlay');
const title = document.getElementById('title');
//const subtitle = document.getElementById('subtitle');

var selectedPlaceName;
var ptHike, featureHike;

function loadPlace(feat) {
    console.log(JSON.stringify(feat))

    const { name, data } = feat.properties;
    const [longitude, latitude] = feat.geometry.coordinates;
    const location = fromLonLat([longitude, latitude]);

    // load data
    hikeSource.clear();
    hikeSource.setUrl(data);
    hikeSource.refresh();

    view.setCenter(location);
    placesListOverlay.classList.add( 'hidden' );

    title.innerHTML = name;
    selectedPlaceName = name;
}

title.addEventListener( 'click', () => {
    placesListOverlay.classList.remove( 'hidden' );
} );

var hikeSource = new VectorSource({
    url: './six_foot_track.json',
    format: new GeoJSON(),

    //url: './mount-colah-to-bobbin-head-loop-via-the-sphinx.gpx',
    //url: './avoca-to-putty-beach.gpx',
    //url: './Wondabyne_27May2023.gpx',
    //url: './CavesBeach_05Aug2023.gpx',
    //url: './bungonia_hike.gpx',
    //url: './Six foot track.gpx',
    //url: './six-foot-track-dec23.gpx',
    //format: new GPX(),
});

var hikeLayer = new VectorLayer({
    title: 'hikes',
    source: hikeSource,
    style: new Style({
        fill: new Fill({
            color: 'rgba(255, 255, 255, 0.6)',
        }),
        stroke: new Stroke({
            color: '#fcba03',
            width: 3,
        }),
    }),
});

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

// hillshade images
const sourceTerrain = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/surfacemap/terrain/{z}/{x}/{y}.img?transp=1`,
  crossOrigin: 'anonymous',
  tileGrid: createXYZ({
    minZoom: 3,
    maxZoom: 15
  }),
});


const elevProfile = new Profile({
    width: 650,
});

hikeSource.on('featuresloadend',function(e) {
    console.log('featuresloadend');

    featureHike = hikeSource.getFeatures()[0];
    elevProfile.setGeometry(featureHike);
    var coord = featureHike.getGeometry().getCoordinates()
    ptHike = new Feature(new Point(coord[0]));
    console.log(coord[0]);
    ptHike.setStyle(pointStyle);
    hikeSource.addFeature(ptHike);
});

// Draw a point on the map when mouse fly over profile
function drawPoint(e) {
  if (!ptHike) return;

  if (e.type=="over"){
    // Show point at coord
    ptHike.setGeometry(new Point(e.coord));
    ptHike.setStyle(pointStyle);
  } else {
    // hide point
    ptHike.setStyle([]);
  }
};
// Show a popup on over
elevProfile.on(["over","out"], function(e) {
  if (e.type=="over") elevProfile.popup(e.coord[2]+" m");
  drawPoint(e);
});

const sourceLocation = new VectorSource();
const locationLayer = new VectorLayer({
  source: sourceLocation,
});

const sourceColorRelief = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/surfacemap/color-relief/{z}/{x}/{y}.img`,
  crossOrigin: 'anonymous',
  tileGrid: createXYZ({
    minZoom: 3,
    maxZoom: 15
  }),
});

const debugLayer = new TileLayer({
  source: new TileDebug({
        projection: 'EPSG:3857',
        tileGrid: createXYZ({
        maxZoom: 21
        })
  }),
  title: 'debug'
});

const hillshadeLayer = new TileLayer({
  source: sourceTerrain,
  opacity: 0.3,
  title: 'hillshade'
});

const basemapLayer = new TileLayer({
    source: new OSM(),
    title: 'Base map'
});

const colormapLayer = new TileLayer({
  source: sourceColorRelief,
  opacity: 0.8,
  title: 'colormap'
});


var ctrInterval = 100;

const view = new View({
  //center: katoomba,
  zoom: 14
});

const labelStyle = new Style({
  text: new Text({
    font: '8px Calibri,sans-serif',
    overflow: true,
    fill: new Fill({
      color: '#000',
    }),
    stroke: new Stroke({
      color: '#fff',
      width: 3,
    }),
  }),
});

const lineStyle = new Style({
  fill: new Fill({
    color: 'rgba(255, 255, 255, 0.6)',
  }),
  stroke: new Stroke({
    color: '#319FD3',
    width: 1,
  }),
});

const pointStyle = new Style({
      image: new Circle({
        radius: 8,
        fill: new Fill({
            color: '#319FD3'
        }),
        stroke: new Stroke({
          color: [255,0,0], 
          width: 2
        })
      })
    });

const style = [lineStyle, labelStyle];

function getContoursUrl(interval) {
    return `${env.contours.proto}://${env.contours.host}:${env.contours.port}/surfacemap/contours/{z}/{x}/{y}.mvt?interval=${interval}`;
}

const contoursLayer = new VectorTileLayer({
  title: 'contours',
  source: new VectorTile({
    url: getContoursUrl(ctrInterval),
    format: new MVT(),
    tileGrid: createXYZ({
        minZoom: 3,
        maxZoom: 15
    }),
    attributions: ['<br>Contours derived from: <a href="https://github.com/tilezen/joerd/blob/master/docs/attribution.md">Licence</a>'],
  }),
  style: function (feature) {
    const label = feature.getProperties()['elevation'].toString() + '\n';
    labelStyle.getText().setText(label);
    return style;
  },
  declutter: true,
});

const attribution = new Attribution({
  collapsible: false,
});

const map = new Map({
  target: 'map',
  layers: [
    basemapLayer,
    colormapLayer,
    hillshadeLayer,
    contoursLayer,
    hikeLayer,
    debugLayer,
    //locationLayer,
  ],
  controls: defaultControls({attribution: false}).extend([attribution]),
  view: view
});

if (places.features.length > 0) {
    loadPlace(places.features[0]);
}

function onClick(id, callback) {
  document.getElementById(id).addEventListener('click', callback);
}

var feature_onHover;
map.on('pointermove', function(evt) {

  feature_onHover = map.forEachFeatureAtPixel(evt.pixel, function(feature, layer) {
    console.log(feature);
    return feature;
  });

  if (feature_onHover == null) {
      return;
  }

  var content = document.getElementById('popup-content');
  var properties = feature_onHover.getProperties()
  console.log(properties.name);
  console.log(JSON.stringify(properties["elevation"]));

  var elevationData = properties["elevation"];
  if (elevationData) {
    var info = document.getElementById('mouse-position');
    var infoText = '<pre>';
    infoText += 'Elevation: ' + JSON.stringify(elevationData)
    infoText += ', '
    infoText += 'Contour interval: ' + ctrInterval + 'm';

    var view = map.getView();
    var coords = view.getCenter();
    var resolution = view.getResolution();
    var projection = view.getProjection();
    var resolutionAtCoords = getPointResolution(projection, resolution, coords);
    infoText += ' . Resolution: ' + resolutionAtCoords.toFixed(2) + 'm';
    infoText += '</pre>';
    info.innerHTML = infoText;

    var coordinate = evt.coordinate;
    content.innerHTML = '<b>Elevation:</b> ' + JSON.stringify(elevationData) + 'm';
    overlay.setPosition(coordinate);
  }
});

var mousePositionControl = new MousePosition({
  coordinateFormat: createStringXY(4),
  projection: 'EPSG:4326'
});

map.addControl(mousePositionControl);

map.addControl(elevProfile);

var container = document.getElementById('popup');
var content = document.getElementById('popup-content');
var closer = document.getElementById('popup-closer');

var overlay = new Overlay({
  element: container,
  autoPan: true,
  autoPanAnimation: {
    duration: 250
  }
});
map.addOverlay(overlay);

closer.onclick = function() {
  overlay.setPosition(undefined);
  closer.blur();
  return false;
};


$("#slider-id").slider({
    value: ctrInterval,
    min: 10,
    max: 500,
    step: 10,
    slide: function(e, ui) {
        ctrInterval = ui.value;

        var info = document.getElementById('mouse-position');
        var infoText = '<pre>';
        infoText += 'Contour interval: ' + ctrInterval + 'm';
        infoText += '</pre>'
        info.innerHTML = infoText;

        let url = getContoursUrl(ctrInterval);
        contoursLayer.getSource().setUrl(url);
    }
});

document.getElementById("checkbox-basemap").addEventListener('change', function() {
  basemapLayer.setVisible(this.checked);
});

document.getElementById("checkbox-contours").addEventListener('change', function() {
  contoursLayer.setVisible(this.checked);
  var ctrlDiv = document.getElementById("slider-id");
  if (this.checked) {
      ctrlDiv.style.visibility='visible';
  } else {
      ctrlDiv.style.visibility='hidden';
  }
});

document.getElementById("checkbox-colormap").addEventListener('change', function() {
  colormapLayer.setVisible(this.checked);
});

document.getElementById("checkbox-hillshade").addEventListener('change', function() {
  hillshadeLayer.setVisible(this.checked);
});

document.getElementById("checkbox-debug").addEventListener('change', function() {
  debugLayer.setVisible(this.checked);
});

document.getElementById("checkbox-basemap").checked = true;
document.getElementById("checkbox-contours").checked = false;
document.getElementById("slider-id").style.visibility='hidden';
document.getElementById("checkbox-colormap").checked = true;
document.getElementById("checkbox-hillshade").checked = true;

debugLayer.setVisible(document.getElementById("checkbox-debug").checked);
basemapLayer.setVisible(document.getElementById("checkbox-basemap").checked);
contoursLayer.setVisible(document.getElementById("checkbox-contours").checked);
colormapLayer.setVisible(document.getElementById("checkbox-colormap").checked);
hillshadeLayer.setVisible(document.getElementById("checkbox-hillshade").checked);

navigator.geolocation.watchPosition(
  function (pos) {
    const coords = [pos.coords.longitude, pos.coords.latitude];
    const accuracy = circular(coords, pos.coords.accuracy);
    sourceLocation.clear(true);
    sourceLocation.addFeatures([
      new Feature(
        accuracy.transform('EPSG:4326', map.getView().getProjection())
      ),
      new Feature(new Point(fromLonLat(coords))),
    ]);
  },
  function (error) {
    alert(`ERROR: ${error.message}`);
  },
  {
    enableHighAccuracy: true,
  }
);

const locate = document.createElement('div');
locate.className = 'ol-control ol-unselectable locate';
locate.innerHTML = '<button title="Locate me">◎</button>';
locate.addEventListener('click', function () {
  if (!sourceLocation.isEmpty()) {
    map.getView().fit(sourceLocation.getExtent(), {
      maxZoom: 15,
      duration: 500,
    });
  }
});
map.addControl(
  new Control({
    element: locate,
  })
);

var ctrlLayerSwitch = new LayerSwitcher({
    // collapsed: false,
    // mouseover: true
});
map.addControl(ctrlLayerSwitch);
ctrlLayerSwitch.on('toggle', function(e) {
    console.log('Collapse layerswitcher', e.collapsed);
});

var btnTerra3d = new Button ({
  html: '<i class="fa fa-map-o"></i>',
  className: "terra3d-btn",
  title: "3D",
  handleClick: function() {
    //info ("hello World!");
    window.location.href = '/terra3d.html';
  }
});
map.addControl(btnTerra3d);

// Show on map over
  var hover = new Hover({ cursor: "pointer", hitTolerance:10 });
  map.addInteraction(hover);
  hover.on("hover", function(e) {
    // Point on the line
    var c = featureHike.getGeometry().getClosestPoint(e.coordinate)
    drawPoint({ type: "over", coord: c });
    // Show profile
    var p = elevProfile.showAt(e.coordinate);
    elevProfile.popup(p[2]+" m");
  });
  hover.on("leave", function(e) {
    elevProfile.popup();
    elevProfile.showAt();
    drawPoint({});
  });

sync(map);

const autoCompleteJS = new autoComplete({
    placeHolder: "Location...",
    threshold: 3,
    searchEngine: "loose",
    data: {
    src: async (query) => {
          try {
                var url = `${env.geocoder.proto}://${env.geocoder.host}:${env.geocoder.port}/geocodeproxy/geocode?q=${query}`;
                const source = await fetch(
                    url,
                    {
                        method: 'GET',
                        headers: {
                            'X-Authorization': `Apikey ${env.geocoder.apikey}`,
                        },
                    }
                );
                // Format data into JSON
                const data = await source.json();
                // Return Fetched data
                console.log(JSON.stringify(data));
                return data.results;
          } catch (error) {
                return error;
          }
        },
        keys: ["formatted"],
        cache: false
    },
    resultItem: {
        highlight: true
    },
    events: {
        input: {
            selection: (event) => {
                const selection = event.detail.selection.value;
                autoCompleteJS.input.value = selection.formatted;
                console.log("selected: " + JSON.stringify(selection));
                
                var feat = {
                    "type": "Feature",
                    "geometry": {"type":"Point", 
                        "coordinates":[]},
                        "properties":{"name":""}
                };
                var lat = selection.geometry.lat;
                var lon = selection.geometry.lng;
                feat.geometry.coordinates = [lon, lat];
                feat.properties['name'] = selection.formatted;
                console.log(JSON.stringify(feat));
                loadPlace(feat);
            }
        }
    }
});
