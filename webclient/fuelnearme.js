import './fuelnearme.css';
import * as env from './env.json';
import Map from 'ol/Map';
import View from 'ol/View';
import {Tile as TileLayer, VectorTile as VectorTileLayer, Image as ImageLayer} from 'ol/layer';
import {TileDebug, OSM, XYZ, VectorTile, Raster} from 'ol/source';
import {GeoJSON, MVT} from 'ol/format';
import {createStringXY} from 'ol/coordinate';
import {fromLonLat, getPointResolution} from 'ol/proj';
import Overlay from 'ol/Overlay';
import {Icon, Circle, Fill, Stroke, Style, Text} from 'ol/style';
import {createXYZ} from 'ol/tilegrid';
import {Attribution, MousePosition, defaults as defaultControls} from 'ol/control';
import sync from 'ol-hashed';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import {circular} from 'ol/geom/Polygon';
import Control from 'ol/control/Control';
import {bbox} from 'ol/loadingstrategy';
import {getCenter} from 'ol/extent';
import {toLonLat} from 'ol/proj';
import {toStringHDMS} from 'ol/coordinate';

// POI
const katoomba = fromLonLat([150.3120553998699, -33.73196775624329])
const asquith = fromLonLat([151.1078953589041,-33.68867268311981])
const sydney = fromLonLat([151.2061999882396,-33.8732161200895])

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

const sourceFuelWatch = new VectorSource({
  format: new GeoJSON(),
  loader: function(extent, resolution, projection, success, failure) {
     var proj = projection.getCode();
     var coord = getCenter(extent)
     var coordLonLat = toLonLat(coord) 
     console.log(coordLonLat); 
     var lon = coordLonLat[0];
     var lat = coordLonLat[1];
     var fuelType = document.querySelector('input[name="fueltype"]:checked').value;
     console.log("fuelType: " + fuelType);
     var url = `${env.nswfuelwatch.proto}://${env.nswfuelwatch.host}:${env.nswfuelwatch.port}/fuelnearme.geojson?lat=${lat}&lon=${lon}&fueltype=${fuelType}`
     console.log(url);
     var xhr = new XMLHttpRequest();
     xhr.open('GET', url);
     var onError = function() {
       sourceFuelWatch.removeLoadedExtent(extent);
       failure();
     }
     xhr.onerror = onError;
     xhr.onload = function() {
       if (xhr.status == 200) {
         var features = sourceFuelWatch.getFormat().readFeatures(xhr.responseText, {
           dataProjection: 'EPSG:4326',
           featureProjection: 'EPSG:3857',
         });
         sourceFuelWatch.addFeatures(features);
         success(features);
       } else {
         onError();
       }
     }
     xhr.send();
   },
   strategy: bbox
});

function pointStyleFunction(feature, resolution) {
  var properties = feature.getProperties()
  var price = properties["price"];
  var textValue = JSON.stringify(price);
  return new Style({
     image: new Circle({
       radius: 14,
       fill: new Fill({
         color: [0, 153, 255, 1],
       }),
       stroke: new Stroke({
         color: [255, 255, 255, 1],
         width: 6,
       }),
     }),
    text: new Text({
        textAlign: "left",
        offsetX: 14,
        text: textValue,
        font: 'bold 16px Calibri,sans-serif',
    }),
     zIndex: Infinity,
  });
}

const fuelWatchLayer = new VectorLayer({
    source: sourceFuelWatch,
    style: pointStyleFunction,
    declutter: true,
    minZoom: 12,
});

const sourceLocation = new VectorSource();
const locationLayer = new VectorLayer({
  source: sourceLocation,
});

const debugLayer = new TileLayer({
    source: new TileDebug({
        projection: 'EPSG:3857',
        tileGrid: createXYZ({
        maxZoom: 21
        })
  })
});

const basemapLayer = new TileLayer({
    source: new OSM()
});

const view = new View({
  center: asquith,
  zoom: 14
});

const attribution = new Attribution({
  collapsible: false,
});

const map = new Map({
  target: 'map',
  layers: [
    basemapLayer,
    fuelWatchLayer,
    debugLayer,
    locationLayer,
  ],
  controls: defaultControls({attribution: false}).extend([attribution]),
  view: view
});

//map.on('singleclick', function (evt) {
//  const coordinate = evt.coordinate;
//  const hdms = toStringHDMS(toLonLat(coordinate));
//
//  content.innerHTML = '<p>You clicked here:</p><code>' + hdms + '</code>';
//  overlay.setPosition(coordinate);
//});

function onClick(id, callback) {
  document.getElementById(id).addEventListener('click', callback);
}

onClick('fly-to-asquith', function() {
  flyTo(asquith, function() {});
});

onClick('fly-to-katoomba', function() {
  flyTo(katoomba, function() {});
});

onClick('fly-to-sydney', function() {
  flyTo(sydney, function() {});
});

function flyTo(location, done) {
    view.setCenter(location);
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

  var fuelType = properties["fueltype"];
  if (fuelType == null) {
      return;
  }

  var price = properties["price"];
  var addr = JSON.stringify(properties["address"]);
  var addrEsc = encodeURIComponent(addr);
  var addrLink = `<a href="https://www.google.com/maps/search/?api=1&query=${addrEsc}">` + addr + '</a>';

  var info = document.getElementById('mouse-position');
  var infoText = '<pre>';
  infoText += 'Fuel: ' + JSON.stringify(properties["fueltype"]);
  infoText += '\n';
  infoText += 'Price: ' + JSON.stringify(price);
  infoText += '\n';
  infoText += `${addrLink}`;
  //infoText += '\n';
  infoText += '</pre>';
  info.innerHTML = infoText;

  var coordinate = evt.coordinate;
  //content.innerHTML = '<b>Price:</b> ' + JSON.stringify(price);
  content.innerHTML = infoText;
  overlay.setPosition(coordinate);
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

function onFuelTypeChange(event) {
    //alert('Checked radio with ID = ' + event.target.id);
    sourceFuelWatch.refresh();
}

document.querySelectorAll("input[name='fueltype']").forEach((input) => {
    input.addEventListener('change', onFuelTypeChange);
});

document.getElementById("checkbox-debug").addEventListener('change', function() {
  debugLayer.setVisible(this.checked);
});

debugLayer.setVisible(document.getElementById("checkbox-debug").checked);

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

sync(map);
