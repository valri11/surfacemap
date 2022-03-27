import './style.css';
import {Map, View} from 'ol';
import TileLayer from 'ol/layer/Tile';
import VectorTileLayer from 'ol/layer/VectorTile';
import OSM from 'ol/source/OSM';
import VectorTile from 'ol/source/VectorTile';
import XYZ from 'ol/source/XYZ';
import GeoJSON from 'ol/format/GeoJSON';
import MVT from 'ol/format/MVT';
import MousePosition from 'ol/control/MousePosition';
import {createStringXY} from 'ol/coordinate';
import * as olProj from 'ol/proj';
import Overlay from 'ol/Overlay';

const source = new XYZ({
    url: 'http://localhost:8000/terra/{z}/{x}/{y}.img' });

const map = new Map({
  target: 'map',
  layers: [
    new TileLayer({
      source: new OSM()
      //source: source
    }),
    new VectorTileLayer({
        source: new VectorTile({
            //url: 'http://localhost:8000/contours/{z}/{x}/{y}.geojson',
            //format: new GeoJSON()
            url: 'http://localhost:8000/contours/{z}/{x}/{y}.mvt',
            format: new MVT()
        })
    }
    ),
  ],
  view: new View({
    //center: olProj.fromLonLat([151.2152272048752, -33.85683819803967]),
    //center: olProj.fromLonLat([150.31161524246588, -33.73080320895877]),
    center: olProj.fromLonLat([74.57950579031711, 42.51248314829303]),
    zoom: 14
  })
});

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
    info.innerHTML = '<p>' + JSON.stringify(properties["elevation"]) + '</p>'

    //overlay.setPosition(evt.coordinate);
     //content.innerHTML = 'HOVER ' + feature_onHover.getProperties().name;
     //container.style.display = 'block';

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

