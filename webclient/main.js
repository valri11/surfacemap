import './main_style.css';
import * as env from './env.json';
import Map from 'ol/Map';
import View from 'ol/View';
import {Tile as TileLayer, VectorTile as VectorTileLayer, Image as ImageLayer} from 'ol/layer';
import {TileDebug, OSM, XYZ, VectorTile, Raster} from 'ol/source';
import {GeoJSON, MVT} from 'ol/format';
import {createStringXY} from 'ol/coordinate';
import {fromLonLat, getPointResolution} from 'ol/proj';
import Overlay from 'ol/Overlay';
import {Fill, Stroke, Style, Text} from 'ol/style';
import {createXYZ} from 'ol/tilegrid';
import {Attribution, MousePosition, defaults as defaultControls} from 'ol/control';

// hillshade images
const sourceTerrain = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/terrain/{z}/{x}/{y}.img`,
  crossOrigin: 'anonymous',
  tileGrid: createXYZ({
    minZoom: 3,
    maxZoom: 15
  }),
});

const sourceColorRelief = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/color-relief/{z}/{x}/{y}.img`,
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
  })
});

const hillshadeLayer = new TileLayer({
  source: sourceTerrain,
  opacity: 0.3,
});

const basemapLayer = new TileLayer({
    source: new OSM()
});

const colormapLayer = new TileLayer({
  source: sourceColorRelief,
  opacity: 0.5,
});

// POI
const kyrg = fromLonLat([74.57950579031711, 42.51248314829303])
const khanTengri = fromLonLat([80.17411914133028, 42.213405765504476])
const katoomba = fromLonLat([150.3120553998699, -33.73196775624329])
const mtDenali = fromLonLat([-151.00726915968875,63.069268194834244])
const pikPobedy = fromLonLat([80.129257551509, 42.03767896555761])
const mtEverest = fromLonLat([86.9251465845193, 27.98955908635046])
const mtOlympus = fromLonLat([22.35011553189942, 40.08838447876729])
const mtKilimanjaro = fromLonLat([37.35554126906301,-3.065881717083569])
const cordilleraBlanca = fromLonLat([-77.5800702637765,-9.169719296932207])
const grandCanyon = fromLonLat([-112.09523569822798,36.10031704536186])
const oahuHawaii = fromLonLat([-157.80960937978762,21.26148763859345])
const mtFuji = fromLonLat([138.73121113691982,35.363529199406074])
const challengerDeep = fromLonLat([142.592522558379, 11.393434778584895])

var ctrInterval = 100;

const view = new View({
  center: katoomba,
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

const style = [lineStyle, labelStyle];

function getContoursUrl(interval) {
    return `${env.contours.proto}://${env.contours.host}:${env.contours.port}/contours/{z}/{x}/{y}.mvt?interval=${interval}`;
}

const contoursLayer = new VectorTileLayer({
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
    contoursLayer,
    colormapLayer,
    hillshadeLayer,
    debugLayer,
  ],
  controls: defaultControls({attribution: false}).extend([attribution]),
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

onClick('fly-to-kilimanjaro', function() {
  flyTo(mtKilimanjaro, function() {});
});

onClick('fly-to-katoomba', function() {
  flyTo(katoomba, function() {});
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

onClick('fly-to-mariana', function() {
  flyTo(challengerDeep, function() {});
});

function flyTo(location, done) {
    view.setCenter(location);
//  const duration = 2000;
//  const zoom = view.getZoom();
//  let parts = 2;
//  let called = false;
//
//  function callback(complete) {
//    contoursLayer.setVisible(false);
//    hillshadeLayer.setVisible(false);
//    --parts;
//    if (called) {
//      return;
//    }
//    if (parts === 0 || !complete) {
//      called = true;
//      var v1 = document.getElementById("checkbox-contours").checked
//      contoursLayer.setVisible(v1);
//      var v3 = document.getElementById("checkbox-hillshade").checked
//      hillshadeLayer.setVisible(v3);
//      done(complete);
//    }
//  }
//  view.animate({
//      center: location,
//      duration: duration,
//    },
//    callback
//  );
//  view.animate({
//      zoom: zoom - 1,
//      duration: duration / 2,
//    }, {
//      zoom: zoom,
//      duration: duration / 2,
//    },
//    callback
//  );
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
    var infoText = '<pre>';
    infoText += 'Elevation: ' + JSON.stringify(properties["elevation"])
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
document.getElementById("checkbox-colormap").checked = true;
document.getElementById("checkbox-hillshade").checked = true;

debugLayer.setVisible(document.getElementById("checkbox-debug").checked);
basemapLayer.setVisible(document.getElementById("checkbox-basemap").checked);
contoursLayer.setVisible(document.getElementById("checkbox-contours").checked);
colormapLayer.setVisible(document.getElementById("checkbox-colormap").checked);
hillshadeLayer.setVisible(document.getElementById("checkbox-hillshade").checked);
