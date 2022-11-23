import './terra3d.css';
import * as env from './env.json';
import Procedural from 'procedural-gl';

// POI
const kyrg = [74.57950579031711, 42.51248314829303]
const khanTengri = [80.17411914133028, 42.213405765504476]
const katoomba = [150.3120553998699, -33.73196775624329]
const uluru = [131.03388514743847, -25.34584297139171]
const mtDenali = [-151.00726915968875,63.069268194834244]
const pikPobedy = [80.129257551509, 42.03767896555761]
const mtEverest = [86.9251465845193, 27.98955908635046]
const mtOlympus = [22.35011553189942, 40.08838447876729]
const mtKilimanjaro = [37.35554126906301,-3.065881717083569]
const cordilleraBlanca = [-77.5800702637765,-9.169719296932207]
const grandCanyon = [-112.09523569822798,36.10031704536186]
const oahuHawaii = [-157.80960937978762,21.26148763859345]
const mtFuji = [138.73121113691982,35.363529199406074]

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

flyTo(katoomba, function() {});

function onClick(id, callback) {
  document.getElementById(id).addEventListener('click', callback);
}

onClick('fly-to-kg', function() {
  flyTo(kyrg, function() {});
});

onClick('fly-to-everest', function() {
  flyTo(mtEverest, function() {});
});

onClick('fly-to-kilimanjaro', function() {
  flyTo(mtKilimanjaro, function() {});
});

onClick('fly-to-katoomba', function() {
  flyTo(katoomba, function() {});
});

onClick('fly-to-uluru', function() {
  flyTo(uluru, function() {});
});

onClick('fly-to-denali', function() {
  flyTo(mtDenali, function() {});
});

onClick('fly-to-cordillera', function() {
  flyTo(cordilleraBlanca, function() {});
});

onClick('fly-to-grand-canyon', function() {
  flyTo(grandCanyon, function() {});
});

onClick('fly-to-pik-pobedy', function() {
  flyTo(pikPobedy, function() {});
});

onClick('fly-to-olympus', function() {
  flyTo(mtOlympus, function() {});
});

onClick('fly-to-khan-tengri', function() {
  flyTo(khanTengri, function() {});
});

onClick('fly-to-oahu', function() {
  flyTo(oahuHawaii, function() {});
});

onClick('fly-to-fuji', function() {
  flyTo(mtFuji, function() {});
});


function flyTo(location, done) {
    Procedural.displayLocation( {
        longitude: location[0],
        latitude: location[1],
        distance: 5000,
    } );
}

navigator.geolocation.watchPosition(
  function (pos) {
    //const coords = [pos.coords.longitude, pos.coords.latitude];
    //const accuracy = circular(coords, pos.coords.accuracy);
    Procedural.setUserLocation(pos);
  },
  function (error) {
    alert(`ERROR: ${error.message}`);
  },
  {
    enableHighAccuracy: true,
  }
);

