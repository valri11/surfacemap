import './style.css';
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
    minZoom: 6,
    maxZoom: 15
  }),
});

const sourceTerra = new XYZ({
  url: `${env.contours.proto}://${env.contours.host}:${env.contours.port}/terra/{z}/{x}/{y}.img`,
  crossOrigin: 'anonymous',
  tileGrid: createXYZ({
    minZoom: 6,
    maxZoom: 15
  }),
});

const raster = new Raster({
  sources: [sourceTerra],
  operationType: 'image',
  operation: shade,
});

const terraLayer = new ImageLayer({
  opacity: 0.3,
  source: raster,
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
const challengerDeep = fromLonLat([142.592522558379, 11.393434778584895])

var ctrInterval = 100;

const view = new View({
  center: kyrg,
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
        minZoom: 6,
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
    debugLayer,
    contoursLayer,
    terraLayer,
    hillshadeLayer
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

onClick('fly-to-mariana', function() {
  flyTo(challengerDeep, function() {});
});

function flyTo(location, done) {
  const duration = 2000;
  const zoom = view.getZoom();
  let parts = 2;
  let called = false;

  function callback(complete) {
    contoursLayer.setVisible(false);
    terraLayer.setVisible(false);
    hillshadeLayer.setVisible(false);
    --parts;
    if (called) {
      return;
    }
    if (parts === 0 || !complete) {
      called = true;
      var v1 = document.getElementById("checkbox-contours").checked
      contoursLayer.setVisible(v1);
      var v2 = document.getElementById("checkbox-terra").checked
      terraLayer.setVisible(v2);
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
  if (this.checked) {
    contoursLayer.setVisible(true);
  } else {
    contoursLayer.setVisible(false);
  }
});

document.getElementById("checkbox-terra").addEventListener('change', function() {
  terraLayer.setVisible(this.checked);
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

function shade(inputs, data) {
  const elevationImage = inputs[0];
  const width = elevationImage.width;
  const height = elevationImage.height;
  const elevationData = elevationImage.data;
  const shadeData = new Uint8ClampedArray(elevationData.length);
  const dp = data.resolution * 2;
  const maxX = width - 1;
  const maxY = height - 1;
  const pixel = [0, 0, 0, 0];
  const twoPi = 2 * Math.PI;
  const halfPi = Math.PI / 2;
  const sunEl = (Math.PI * data.sunEl) / 180;
  const sunAz = (Math.PI * data.sunAz) / 180;
  const cosSunEl = Math.cos(sunEl);
  const sinSunEl = Math.sin(sunEl);
  let pixelX,
    pixelY,
    x0,
    x1,
    y0,
    y1,
    offset,
    z0,
    z1,
    dzdx,
    dzdy,
    slope,
    aspect,
    cosIncidence,
    scaled;
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
  for (pixelY = 0; pixelY <= maxY; ++pixelY) {
    y0 = pixelY === 0 ? 0 : pixelY - 1;
    y1 = pixelY === maxY ? maxY : pixelY + 1;
    for (pixelX = 0; pixelX <= maxX; ++pixelX) {
      x0 = pixelX === 0 ? 0 : pixelX - 1;
      x1 = pixelX === maxX ? maxX : pixelX + 1;

      // determine elevation for (x0, pixelY)
      offset = (pixelY * width + x0) * 4;
      pixel[0] = elevationData[offset];
      pixel[1] = elevationData[offset + 1];
      pixel[2] = elevationData[offset + 2];
      pixel[3] = elevationData[offset + 3];
      z0 = data.vert * calculateElevation(pixel);

      // determine elevation for (x1, pixelY)
      offset = (pixelY * width + x1) * 4;
      pixel[0] = elevationData[offset];
      pixel[1] = elevationData[offset + 1];
      pixel[2] = elevationData[offset + 2];
      pixel[3] = elevationData[offset + 3];
      z1 = data.vert * calculateElevation(pixel);

      dzdx = (z1 - z0) / dp;

      // determine elevation for (pixelX, y0)
      offset = (y0 * width + pixelX) * 4;
      pixel[0] = elevationData[offset];
      pixel[1] = elevationData[offset + 1];
      pixel[2] = elevationData[offset + 2];
      pixel[3] = elevationData[offset + 3];
      z0 = data.vert * calculateElevation(pixel);

      // determine elevation for (pixelX, y1)
      offset = (y1 * width + pixelX) * 4;
      pixel[0] = elevationData[offset];
      pixel[1] = elevationData[offset + 1];
      pixel[2] = elevationData[offset + 2];
      pixel[3] = elevationData[offset + 3];
      z1 = data.vert * calculateElevation(pixel);

      dzdy = (z1 - z0) / dp;

      slope = Math.atan(Math.sqrt(dzdx * dzdx + dzdy * dzdy));

      aspect = Math.atan2(dzdy, -dzdx);
      if (aspect < 0) {
        aspect = halfPi - aspect;
      } else if (aspect > halfPi) {
        aspect = twoPi - aspect + halfPi;
      } else {
        aspect = halfPi - aspect;
      }

      cosIncidence =
        sinSunEl * Math.cos(slope) +
        cosSunEl * Math.sin(slope) * Math.cos(sunAz - aspect);

      offset = (pixelY * width + pixelX) * 4;
      scaled = 255 * cosIncidence;
      shadeData[offset] = scaled;
      shadeData[offset + 1] = scaled;
      shadeData[offset + 2] = scaled;
      shadeData[offset + 3] = elevationData[offset + 3];
    }
  }

  return {data: shadeData, width: width, height: height};
}

const controlIds = ['vert', 'sunEl', 'sunAz'];
const controls = {};
controlIds.forEach(function (id) {
  const control = document.getElementById(id);
  const output = document.getElementById(id + 'Out');
  const listener = function () {
    output.innerText = control.value;
    raster.changed();
  };
  control.addEventListener('input', listener);
  control.addEventListener('change', listener);
  output.innerText = control.value;
  controls[id] = control;
});

raster.on('beforeoperations', function (event) {
  // the event.data object will be passed to operations
  const data = event.data;
  data.resolution = event.resolution;
  for (const id in controls) {
    data[id] = Number(controls[id].value);
  }
});

document.getElementById("checkbox-basemap").checked = true;
document.getElementById("checkbox-contours").checked = true;
document.getElementById("checkbox-terra").checked = false;
document.getElementById("checkbox-hillshade").checked = true;
basemapLayer.setVisible(true);
contoursLayer.setVisible(true);
terraLayer.setVisible(false);
hillshadeLayer.setVisible(true);
