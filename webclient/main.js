import './style.css';
import {
  Map,
  View
} from 'ol';
import TileLayer from 'ol/layer/Tile';
import VectorTileLayer from 'ol/layer/VectorTile';
import OSM from 'ol/source/OSM';
import VectorTile from 'ol/source/VectorTile';
import XYZ from 'ol/source/XYZ';
import GeoJSON from 'ol/format/GeoJSON';
import MVT from 'ol/format/MVT';
import MousePosition from 'ol/control/MousePosition';
import {
  createStringXY
} from 'ol/coordinate';
import * as olProj from 'ol/proj';
import Overlay from 'ol/Overlay';

const source = new XYZ({
  url: 'http://localhost:8000/terra/{z}/{x}/{y}.img'
});

const kyrg = olProj.fromLonLat([74.57950579031711, 42.51248314829303])
const mtEverest = olProj.fromLonLat([86.9251465845193, 27.98955908635046])
const katoomba = olProj.fromLonLat([150.3120553998699, -33.73196775624329])
const grandCanyonUSA = olProj.fromLonLat([-118.12954343868806, 34.22960585491841])
const pikPobedy = olProj.fromLonLat([80.129257551509, 42.03767896555761])
const mtOlympus = olProj.fromLonLat([22.35011553189942, 40.08838447876729])
const khanTengri = olProj.fromLonLat([80.17411914133028, 42.213405765504476])
const challengerDeep = olProj.fromLonLat([142.592522558379, 11.393434778584895])

const view = new View({
  center: kyrg,
  zoom: 14
});

const contours = new VectorTileLayer({
  source: new VectorTile({
    //url: 'http://localhost:8000/contours/{z}/{x}/{y}.geojson',
    //format: new GeoJSON()
    url: 'http://localhost:8000/contours/{z}/{x}/{y}.mvt',
    format: new MVT()
  })
});

const map = new Map({
  target: 'map',
  layers: [
    new TileLayer({
      source: new OSM()
      //source: source
    }),
    contours
  ],
  view: view
});

function onClick(id, callback) {
  document.getElementById(id).addEventListener('click', callback);
}

onClick('fly-to-kg', function() {
  flyTo(kyrg, function() {});
});

onClick('fly-to-everest', function() {
  flyTo(mtEverest, function() {});
});

onClick('fly-to-katoomba', function() {
  flyTo(katoomba, function() {});
});

onClick('fly-to-grand-canyon', function() {
  flyTo(grandCanyonUSA, function() {});
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

onClick('fly-to-mariana', function() {
  flyTo(challengerDeep, function() {});
});

function flyTo(location, done) {
  const duration = 2000;
  const zoom = view.getZoom();
  let parts = 2;
  let called = false;

  function callback(complete) {
    contours.setVisible(false);
    --parts;
    if (called) {
      return;
    }
    if (parts === 0 || !complete) {
      called = true;
      contours.setVisible(true);
      done(complete);
    }
  }
  view.animate({
      center: location,
      duration: duration,
    },
    callback
  );
  view.animate({
      zoom: zoom - 1,
      duration: duration / 2,
    }, {
      zoom: zoom,
      duration: duration / 2,
    },
    callback
  );
}

var feature_onHover;
map.on('pointermove', function(evt) {

  feature_onHover = map.forEachFeatureAtPixel(evt.pixel, function(feature, layer) {
    console.log(feature);
    return feature;
  });

  if (feature_onHover) {
    var content = document.getElementById('popup-content');
    var properties = feature_onHover.getProperties()
    console.log(properties.name);
    console.log(JSON.stringify(properties["elevation"]));

    var info = document.getElementById('mouse-position');
    info.innerHTML = '<pre>' + 'Elevation: ' + JSON.stringify(properties["elevation"]) + '</pre>'

    var coordinate = evt.coordinate;

    content.innerHTML = '<b>Elevation:</b> ' + JSON.stringify(properties["elevation"]) + 'm';
    overlay.setPosition(coordinate);

  } else {
    //container.style.display = 'none';
  }
});


var mousePositionControl = new MousePosition({
  coordinateFormat: createStringXY(4),
  projection: 'EPSG:4326'
});

map.addControl(mousePositionControl);

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