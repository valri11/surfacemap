import './sealevel.css';
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

const sourceTerra = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/terra/{z}/{x}/{y}.img`,
  crossOrigin: 'anonymous',
  interpolate: false,
  tileGrid: createXYZ({
    minZoom: 3,
    maxZoom: 15
  }),
});

const raster = new Raster({
  sources: [sourceTerra],
  //operationType: 'image',
  operation: flood,
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

const sealevelLayer = new ImageLayer({
    opacity: 0.6,
    source: raster,
});


const newYork = fromLonLat([-74.04442672993127,40.69010807133021])
const london = fromLonLat([-0.12467245895210548,51.50101981028784])
const paris = fromLonLat([2.3360465678907536,48.85928539255598])
const venice = fromLonLat([12.340694942695933,45.43544087127655])
const operaHouse = fromLonLat([151.2153396126718,-33.85659727934901])
const oahuHawaii = fromLonLat([-157.80960937978762,21.26148763859345])

const lismore = fromLonLat([153.27707525263946,-28.80607911799792])
const windsor = fromLonLat([150.822676436187,-33.60364397111745])

var ctrInterval = 100;

const view = new View({
  center: operaHouse,
  zoom: 15
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
    hillshadeLayer,
    sealevelLayer,
    debugLayer,
  ],
  controls: defaultControls({attribution: false}).extend([attribution]),
  view: view
});

function onClick(id, callback) {
  document.getElementById(id).addEventListener('click', callback);
}

onClick('fly-to-sydney', function() {
  flyTo(operaHouse, function() {});
});

onClick('fly-to-newyork', function() {
  flyTo(newYork, function() {});
});

onClick('fly-to-london', function() {
  flyTo(london, function() {});
});

onClick('fly-to-paris', function() {
  flyTo(paris, function() {});
});

onClick('fly-to-venice', function() {
  flyTo(venice, function() {});
});

onClick('fly-to-hawaii', function() {
  flyTo(oahuHawaii, function() {});
});

onClick('fly-to-windsor', function() {
  flyTo(windsor, function() {});
});

onClick('fly-to-lismore', function() {
  flyTo(lismore, function() {});
});

function flyTo(location, done) {
  const duration = 2000;
  const zoom = view.getZoom();
  let parts = 2;
  let called = false;

  function callback(complete) {
    contoursLayer.setVisible(false);
    hillshadeLayer.setVisible(false);
    --parts;
    if (called) {
      return;
    }
    if (parts === 0 || !complete) {
      called = true;
      var v1 = document.getElementById("checkbox-contours").checked
      contoursLayer.setVisible(v1);
      var v3 = document.getElementById("checkbox-hillshade").checked
      hillshadeLayer.setVisible(v3);
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
  if (this.checked) {
    contoursLayer.setVisible(true);
  } else {
    contoursLayer.setVisible(false);
  }
});

document.getElementById("checkbox-sealevel").addEventListener('change', function() {
  sealevelLayer.setVisible(this.checked);
});

document.getElementById("checkbox-hillshade").addEventListener('change', function() {
  hillshadeLayer.setVisible(this.checked);
});

var showDebug = document.getElementById("checkbox-debug").checked
debugLayer.setVisible(showDebug);

document.getElementById("checkbox-debug").addEventListener('change', function() {
  if (this.checked) {
    debugLayer.setVisible(true);
  } else {
    debugLayer.setVisible(false);
  }
});


function flood(pixels, data) {
  const pixel = pixels[0];
    function calculateElevation(pixel) {
        // The method used to extract elevations from the DEM.
        // In this case the format used is
        // red + green * 2 + blue * 3
        //
        // Other frequently used methods include the Mapbox format
        // (red * 256 * 256 + green * 256 + blue) * 0.1 - 10000
        // and the Terrarium format
        // (red * 256 + green + blue / 256) - 32768
        //
        //return pixel[0] + pixel[1] * 2 + pixel[2] * 3;
        return (pixel[0] * 256 + pixel[1] + pixel[2] / 256) - 32768;
    }
  if (pixel[3]) {
    const height = calculateElevation(pixel);
    if (height <= data.level) {
      pixel[0] = 134;
      pixel[1] = 203;
      pixel[2] = 249;
      pixel[3] = 255;
    } else {
      pixel[3] = 0;
    }
  }
  return pixel;
}

const control = document.getElementById('level');
const output = document.getElementById('output');
const listener = function () {
  output.innerText = control.value;
  raster.changed();
};
control.addEventListener('input', listener);
control.addEventListener('change', listener);
output.innerText = control.value;

document.getElementById("checkbox-basemap").checked = true;
document.getElementById("checkbox-contours").checked = false;
document.getElementById("checkbox-sealevel").checked = true;
document.getElementById("checkbox-hillshade").checked = true;
basemapLayer.setVisible(true);
contoursLayer.setVisible(false);
sealevelLayer.setVisible(true);
hillshadeLayer.setVisible(true);

raster.on('beforeoperations', function (event) {
  event.data.level = control.value;
});
