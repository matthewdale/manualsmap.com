"use strict";

var canvi = new Canvi({
    navbar: ".canvi-navbar",
    content: ".canvi-content",
    pushContent: false,
    width: "18.5em",
});

mapkit.init({
    authorizationCallback: function (done) {
        fetch("/token")
            .then(response => response.json())
            .then(result => {
                done(result.token)
            });
    }
});

var map = new mapkit.Map("map", {
    tracksUserLocation: true,
    showsUserLocationControl: true,
});


//////// Search ////////
function submitSearch() {
    search($("#searchQuery").val());
}

function search(query) {
    let searcher = new mapkit.Search({ region: map.region });
    searcher.search(query, function (error, data) {
        if (error) {
            // Handle search error
            return;
        }
        let annotations = data.places.map(function (place) {
            let annotation = new mapkit.MarkerAnnotation(place.coordinate);
            annotation.title = place.name;
            annotation.subtitle = place.formattedAddress;
            annotation.color = "#9B6134";
            return annotation;
        });
        map.showItems(annotations);
    });
}

//////// Add Car ////////
var CALLOUT_OFFSET = new DOMPoint(-148, -78);
var newCarAnnotation = null;
function addCar() {
    if (newCarAnnotation) {
        map.removeAnnotation(newCarAnnotation);
    }
    let form = $("#addCarForm").clone();
    form.on("submit", function (event) {
        // TODO: Form validation.
        // TODO: Submit data.
        let form = event.target;
        let data = {
            car: {
                year: Number(form["year"].value),
                brand: form["brand"].value,
                model: form["model"].value,
                trim: form["trim"].value,
                color: form["color"].value,
            },
            licenseState: form["licenseState"].value,
            licensePlate: form["licensePlate"].value,
            latitude: newCarAnnotation.coordinate.latitude,
            longitude: newCarAnnotation.coordinate.longitude,
        };
        console.log(data);
        console.log(JSON.stringify(data));
        let options = {
            method: "POST",
            body: JSON.stringify(data),
            headers: {
                "Content-Type": "application/json",
            },
        };
        fetch("/cars", options)
            .then(res => res.json())
            .then(result => {
                console.log(result)
                // TODO: What about on failure?
                fetchVisibleOverlays();
            });

        map.removeAnnotation(newCarAnnotation);

        newCarAnnotation = null;
        return false;
    })
    let callout = {
        calloutElementForAnnotation: function (annotation) {
            let div = document.createElement("div");
            div.className = "landmark";

            div.appendChild(form.get(0));

            return div;
        },
        calloutAnchorOffsetForAnnotation: function (annotation, element) {
            return CALLOUT_OFFSET;
        },
        calloutAppearanceAnimationForAnnotation: function (annotation) {
            return "scale-and-fadein .4s 0 1 normal cubic-bezier(0.4, 0, 0, 1.5)";
        },
    };
    newCarAnnotation = new mapkit.MarkerAnnotation(map.center, {
        draggable: true,
        title: "Click and hold to drag",
        callout: callout,
    });
    map.addAnnotation(newCarAnnotation);
}

map.addEventListener("drag-end", function (event) {
    console.log("drag-end");
    console.log(event.annotation.coordinate);
    // TODO: Highlight target overlay.
});


//////// Display Cars ////////
function displayCars(cars) {
    let elements = cars.map(car => {
        let div = $("#carDisplay div").clone();
        div.find("#year").text(car.year);
        div.find("#brand").text(car.brand);
        div.find("#model").text(car.model);
        div.find("#trim").text(car.trim);
        div.find("#color").text(car.color);
        div.find("#imageLink").prop("href", car.image_url);
        div.find("#image").prop("src", car.image_url);

        return div;
    });

    $("#carsInfo").html("");
    $("#carsInfo").append(elements);

    // TODO: Necessary?
    canvi.open();
    canvi._removeOverlay();
}

// TODO: Necessary?
function hideCars() {
    $("#carsInfo").html("");
    canvi.close();
}


//////// Display Map Blocks ////////
var overlayStyle = new mapkit.Style({
    strokeColor: "#F00",
    strokeOpacity: .2,
    lineWidth: 1,
    lineJoin: "round",
    lineDash: [2, 2, 6, 2, 6, 2]
});
function buildOverlays(mapBlocks) {
    return mapBlocks.map(block => {
        // Determine if the offset should be positive or negative so
        // that the magnitude of the sum is always greater (i.e.
        // farther away from zero. This works in conjunction with
        // truncating the car longitude and latitude, which always
        // rounds toward zero.
        let latOffset = 0.00999 * Math.sign(block.latitude);
        let longOffset = 0.00999 * Math.sign(block.longitude);
        let overlay = new mapkit.PolygonOverlay([
            new mapkit.Coordinate(block.latitude, block.longitude),
            new mapkit.Coordinate(block.latitude, block.longitude + longOffset),
            new mapkit.Coordinate(block.latitude + latOffset, block.longitude + longOffset),
            new mapkit.Coordinate(block.latitude + latOffset, block.longitude),
        ], {
            style: overlayStyle,
            visible: true,
            enabled: true,
            data: {
                id: block.id,
                cars: null,
            },
        });
        overlay.addEventListener("select", function (event) {
            // TODO: Highlight selected overlay.
            if (event.target.data.cars) {
                displayCars(event.target.data.cars);
                return;
            }
            fetch(`/mapblocks/${event.target.data.id}/cars`)
                .then(response => response.json())
                .then(result => {
                    event.target.data.cars = result.cars;
                    displayCars(result.cars);
                });
        });
        overlay.addEventListener("deselect", function (event) {
            // Figure out how to cancel the previous request or
            // prevent the info from being displayed.
            // hideCars();
        });
        return overlay;
    });
}
function fetchVisibleOverlays() {
    // If we're going to fetch too many blocks, skip it.
    if (map.region.span.latitudeDelta >= maxSpan || map.region.span.longitudeDelta >= maxSpan) {
        map.removeOverlays(overlays);
        overlays = [];
        return;
    }

    let region = map.region.toBoundingRegion();
    let minLatitude = region.southLatitude;
    let minLongitude = region.westLongitude;
    let maxLatitude = region.northLatitude;
    let maxLongitude = region.eastLongitude;
    let query = `min_latitude=${minLatitude}&min_longitude=${minLongitude}&max_latitude=${maxLatitude}&max_longitude=${maxLongitude}`
    fetch("/mapblocks?" + query)
        .then(response => response.json())
        .then(result => {
            map.removeOverlays(overlays);
            overlays = buildOverlays(result.map_blocks);
            map.addOverlays(overlays);
        });
}

// The maximum longitude or latitude span to display overlays/
const maxSpan = 0.6;
var overlays = [];
map.addEventListener("region-change-end", function (event) {
    fetchVisibleOverlays();
});

canvi.open();
canvi._removeOverlay();
