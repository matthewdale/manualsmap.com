"use strict";

var canvi = new Canvi({
    navbar: ".canvi-navbar",
    content: ".canvi-content",
    pushContent: false,
    width: "18.5em",
});

mapkit.init({
    authorizationCallback: function (done) {
        fetch("/mapkit/token")
            .then(response => response.json())
            .then(result => {
                done(result.token)
            });
    }
});

function generateSignature(callback, parameters) {
    let strParameters = {};
    for (let key in parameters) {
        strParameters[key] = parameters[key].toString();
    }
    let options = {
        method: "POST",
        body: JSON.stringify({ parameters: strParameters }),
        headers: {
            "Content-Type": "application/json",
        },
    };
    fetch("/images/signature", options)
        .then(res => res.json())
        .then(result => {
            // TODO: What about on failure?
            callback(result.signature);
        });
}

var uploadWidget = cloudinary.createUploadWidget({
    cloudName: "dawfgqsur",
    apiKey: 263238496553624, 
    uploadPreset: "manualsmap_com",
    uploadSignature: generateSignature,
    sources: ["local", "url", "camera"],
}, (error, result) => {
    // TODO: Figure out how to get this result into the POST /cars form.
    if (!error && result && result.event === "success") {
        console.log("Done! Here is the image info: ", result.info);
    }
})

var map = new mapkit.Map("map", {
    // TODO: Is there any way to get rid of the user location overlay?
    // tracksUserLocation: true,
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
            // TODO: Handle search error.
            return;
        }
        let annotations = data.places.map(function (place) {
            let annotation = new mapkit.MarkerAnnotation(place.coordinate);
            annotation.title = place.name;
            annotation.subtitle = place.formattedAddress;
            annotation.color = "#9B6134";
            return annotation;
        });
        // TODO: Remove other search result markers.
        // TODO: Set maximum zoom on search.
        map.showItems(annotations);
    });
}


//////// Add Car ////////
function renderRecaptcha() {
    grecaptcha.render("recaptcha", {
        "sitekey": "6Ld5i-MUAAAAAMAIZ1my_sonpYAECKc4UIdiIvhQ",
        "theme": "light",
    });
}

var CALLOUT_OFFSET = new DOMPoint(-148, -78);
var newCarAnnotation = null;
function addCar() {
    if (newCarAnnotation) {
        map.removeAnnotation(newCarAnnotation);
    }
    let form = $("#addCarFormTemplate").clone();
    form.prop("id", "addCarForm");

    // Add and render reCAPTCHA checkbox.
    let recaptcha = document.createElement("div");
    recaptcha.setAttribute("id", "recaptcha");
    let script = document.createElement("script");
    script.appendChild(document.createTextNode("renderRecaptcha();"));
    form.find("#recaptcha").before(recaptcha);
    form.find("#recaptcha").before(script);

    form.find("#cloudinary").on("click", function () {
        uploadWidget.open();
        return false;
    });

    form.on("submit", function (event) {
        // TODO: Form validation.
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
            recaptcha: grecaptcha.getResponse(),
        };
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

var addCarOverlay = null;
map.addEventListener("dragging", function (event) {
    if (addCarOverlay) {
        map.removeOverlay(addCarOverlay);
    }
    // TODO: Change color of add car overlay.
    addCarOverlay = mapBlockOverlay(
        truncate(event.coordinate.latitude, 2),
        truncate(event.coordinate.longitude, 2));
    map.addOverlay(addCarOverlay);
});

// Based on https://code-examples.net/en/q/3fe40a
function truncate(number, digits) {
    var re = new RegExp('^-?\\d+(?:\.\\d{0,' + (digits || -1) + '})?');
    return Number(number.toString().match(re)[0]);
};

//////// Display Cars ////////
function displayCars(cars) {
    let elements = cars.map(car => {
        let div = $("#carDisplay div").clone();
        div.find("#year").text(car.year);
        div.find("#brand").text(car.brand);
        div.find("#model").text(car.model);
        div.find("#trim").text(car.trim);
        div.find("#color").text(car.color);
        div.find("#imageLink").prop("href", car.imageUrl);
        div.find("#image").prop("src", car.imageUrl);

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
    fillColor: "#007bff",
    fillOpacity: 0.3,
    strokeColor: "#ff0000",
    strokeOpacity: 0.5,
    lineWidth: 1,
    lineJoin: "round",
    lineDash: [2, 2, 6, 2, 6, 2]
});
function mapBlockOverlay(latitude, longitude) {
    // Determine if the offset should be positive or negative so
    // that the magnitude of the sum is always greater (i.e.
    // farther away from zero. This works in conjunction with
    // truncating the car longitude and latitude, which always
    // rounds toward zero.
    let latOffset = 0.00999 * Math.sign(latitude);
    let longOffset = 0.00999 * Math.sign(longitude);
    return new mapkit.PolygonOverlay([
        new mapkit.Coordinate(latitude, longitude),
        new mapkit.Coordinate(latitude, longitude + longOffset),
        new mapkit.Coordinate(latitude + latOffset, longitude + longOffset),
        new mapkit.Coordinate(latitude + latOffset, longitude),
    ], {
        style: overlayStyle,
        visible: true,
        enabled: true,
    });
}

function buildOverlays(mapBlocks) {
    return mapBlocks.map(block => {
        let overlay = mapBlockOverlay(block.latitude, block.longitude);
        overlay.data = { id: block.id };
        overlay.addEventListener("select", function (event) {
            // TODO: Highlight selected overlay.
            fetch(`/mapblocks/${event.target.data.id}/cars`)
                .then(response => response.json())
                .then(result => {
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
            overlays = buildOverlays(result.mapBlocks);
            map.addOverlays(overlays);
        });
}

// The maximum longitude or latitude span to display overlays/
const maxSpan = 0.7;
var overlays = [];
map.addEventListener("region-change-end", function (event) {
    fetchVisibleOverlays();
});

canvi.open();
canvi._removeOverlay();
