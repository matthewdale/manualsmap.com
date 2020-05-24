"use strict";

var canvi = new Canvi({
    navbar: ".canvi-navbar",
    content: ".canvi-content",
    pushContent: false,
    width: "80%",
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

// See http://www.carqueryapi.com/
var carquery = new CarQuery();
carquery.init();
carquery.setFilters({ sold_in_us: true });
carquery.initYearMakeModelTrim("car-years", "car-makes", "car-models", "car-model-trims");



var map = new mapkit.Map("map", {
    // TODO: Is there any way to get rid of the user location overlay?
    showsUserLocation: false,
    tracksUserLocation: true,
    showsUserLocationControl: true,
});

function handleErrors(response) {
    if (!response.ok) {
        throw Error(response.statusText);
    }
    return response;
}

function addCarModal(options) {
    $("#addCarModal").modal(options);
}

function whatModal(options) {
    $("#whatModal").modal(options);
}

function privacyModal(options) {
    $("#privacyModal").modal(options);
}

//////// Search ////////
function submitSearch() {
    let searchInput = $("#searchInput");
    search(searchInput.val());
    searchInput.blur();
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
var addCarOverlay = null;
function drawAddCarOverlay(latitude, longitude) {
    if (addCarOverlay) {
        map.removeOverlay(addCarOverlay);
    }
    addCarOverlay = mapBlockOverlay(
        segmentCoordinate(latitude),
        segmentCoordinate(longitude),
        "#00FF7B");
    map.addOverlay(addCarOverlay);
}

function isVisible(annotation) {
    let region = map.region.toBoundingRegion();
    let coordinate = annotation.coordinate;

    return coordinate.latitude <= region.northLatitude &&
        coordinate.latitude >= region.southLatitude &&
        coordinate.longitude <= region.eastLongitude &&
        coordinate.longitude >= region.westLongitude;
}

var addCarAnnotation = null;
function placeOnMap() {
    if (addCarAnnotation) {
        return;
    }
    addCarAnnotation = new mapkit.MarkerAnnotation(map.center, {
        draggable: true,
        calloutEnabled: false,
        title: "Click and hold to position",
        titleVisibility: "visible",
    });
    addCarAnnotation.addEventListener("dragging", function (event) {
        drawAddCarOverlay(
            event.coordinate.latitude,
            event.coordinate.longitude);
    });
    addCarAnnotation.addEventListener("drag-end", function (event) {
        $("#addCar #latitude").val(event.target.coordinate.latitude);
        $("#addCar #longitude").val(event.target.coordinate.longitude);
        addCarModal("show");
    });
    map.addAnnotation(addCarAnnotation);

    drawAddCarOverlay(
        addCarAnnotation.coordinate.latitude,
        addCarAnnotation.coordinate.longitude);
}

var cloudinaryUploadInfo;
const cloudinaryCloudName = "dawfgqsur";
const cloudinaryUploadPreset = "manualsmap_com";
function addImage(token) {
    let timestamp = Math.round((new Date()).getTime() / 1000);
    let options = {
        method: "POST",
        body: JSON.stringify({
            parameters: {
                source: "uw",
                timestamp: timestamp.toString(),
                upload_preset: cloudinaryUploadPreset,
            },
            recaptcha: token,
        }),
        headers: {
            "Content-Type": "application/json",
        },
    };
    fetch("/images/signature", options)
        .then(res => {
            handleErrors(res);
            return res.json();
        })
        .then(result => {
            var widget = cloudinary.createUploadWidget({
                cloudName: cloudinaryCloudName,
                apiKey: 263238496553624,
                uploadPreset: cloudinaryUploadPreset,
                uploadSignature: result.signature,
                uploadSignatureTimestamp: timestamp,
                sources: ["local", "url", "camera"],
                multiple: false,
            }, (error, result) => {
                if (!error && result && result.event === "success") {
                    console.log("Done! Here is the image info: ", result.info);
                    cloudinaryUploadInfo = result.info;

                    let form = $("#addCar");
                    form.find("#image").prop("src", cloudinaryUploadInfo.secure_url);

                    let addImage = form.find("#addImage");
                    addImage.hide();

                    let removeImage = form.find("#removeImage");
                    removeImage.show();
                    removeImage.on("click", function () {
                        deleteImage(cloudinaryUploadInfo.delete_token);
                        cloudinaryUploadInfo = null;
                        form.find("#image").prop("src", "");
                        form.find("#addImage").show();
                        form.find("#removeImage").hide();
                    });
                }
            });
            widget.open();
        }).catch(error => {
            alert("Failed to sign upload: " + error);
        });
}

function deleteImage(token) {
    let options = {
        method: "POST",
        body: "token=" + encodeURIComponent(token),
        headers: {
            "Content-Type": "application/x-www-form-urlencoded",
        },
    };
    fetch(`https://api.cloudinary.com/v1_1/${cloudinaryCloudName}/delete_by_token`, options);
}

function resetAddCarValidation() {
    let form = $("#addCar");
    let formEl = form.get(0);
    form.serializeArray().forEach(function (field, _) {
        if (!(field.name in formEl)) {
            return;
        }
        formEl[field.name].classList.remove("is-invalid");
    })
}

function resetAddCar() {
    resetAddCarValidation();

    let form = $("#addCar");
    let formEl = form.get(0);
    [
        "year",
        "make",
        "model",
        "trim",
        "color",
    ].forEach(function (field, _) {
        formEl[field].selectedIndex = 0;
    });
    [
        "latitude",
        "longitude",
    ].forEach(function (field, _) {
        formEl[field].value = "";
    });

    form.find("#image").prop("src", "");
    form.find("#addImage").show();
    form.find("#removeImage").hide();

    if (addCarOverlay) {
        map.removeOverlay(addCarOverlay);
        addCarOverlay = null;
    }
    if (addCarAnnotation) {
        map.removeAnnotation(addCarAnnotation);
        addCarAnnotation = null;
    }
}

var postCarsValidator;
fetch("/cars/schema")
    .then(res => {
        handleErrors(res);
        return res.json();
    }).then(result => {
        let ajv = new Ajv({ allErrors: true });
        postCarsValidator = ajv.compile(result);
    }).catch(error => {
        alert("Failed to fetch POST /cars schema: " + error);
    });

function validateAddCar(data) {
    var valid = postCarsValidator(data);
    if (!valid) {
        return JSON.parse(JSON.stringify(postCarsValidator.errors));
    }
    return []
}

function getSelectedText(el) {
    return el[el.selectedIndex].innerText;
}

function submitCar(token) {
    let form = $("#addCar");
    let formEl = form.get(0);
    let trim = getSelectedText(formEl["trim"]);
    // The default selection for the CarQuery data is "None". If the user
    // selects "None", change it to an empty string instead.
    if (trim.toLowerCase() == "none") {
        trim = "";
    }
    let data = {
        year: Number(getSelectedText(formEl["year"])),
        make: getSelectedText(formEl["make"]),
        model: getSelectedText(formEl["model"]),
        trim: getSelectedText(formEl["trim"]),
        color: getSelectedText(formEl["color"]),
        recaptcha: token,
    };
    if (formEl["latitude"].value) {
        data.latitude = Number(formEl["latitude"].value);
    }
    if (formEl["longitude"].value) {
        data.longitude = Number(formEl["longitude"].value);
    }
    if (cloudinaryUploadInfo) {
        data.cloudinaryPublicId = cloudinaryUploadInfo.public_id;
    }

    let errors = validateAddCar(data);
    if (errors.length) {
        console.log(errors);
        resetAddCarValidation();
        errors.map(err => {
            switch (err.keyword) {
                case "required":
                    return err.params.missingProperty;
                default:
                    return err.dataPath.slice(1);
            }
        }).forEach(function (field, _) {
            if (!(field in formEl)) {
                return;
            }
            formEl[field].classList.add("is-invalid");
        });
        return false;
    }

    let options = {
        method: "POST",
        body: JSON.stringify(data),
        headers: {
            "Content-Type": "application/json",
        },
    };
    fetch("/cars", options)
        .then(res => {
            handleErrors(res);
            return res.json();
        }).then(result => {
            addCarModal("hide");
            resetAddCar();

            displayCars(result.mapBlockId);
            fetchVisibleOverlays();
        }).catch(error => {
            alert("Failed to add car: " + error);
        });
}

const mapBlockSize = 0.05;
function segmentCoordinate(coordinate) {
    return truncate(truncate(coordinate / mapBlockSize, 0) * mapBlockSize, 2);
}

// Based on https://code-examples.net/en/q/3fe40a
function truncate(number, digits) {
    var re = new RegExp('^-?\\d+(?:\.\\d{0,' + digits + '})?');
    return Number(number.toString().match(re)[0]);
};


//////// Display Cars ////////
function displayCars(mapBlockId) {
    let container = $("#cars");
    container.html("");
    canvi.open();

    fetch(`/mapblocks/${mapBlockId}/cars`)
        .then(res => {
            handleErrors(res);
            return res.json();
        })
        .then(result => {
            // Convert each car into a Bootstrap card.
            let cards = result.cars.map(car => {
                let div = $("#carTemplate .car").clone();
                div.find("#year").text(car.year);
                div.find("#make").text(car.make);
                div.find("#model").text(car.model);
                div.find("#trim").text(car.trim);
                div.find("#color").text(car.color);

                // TODO: Handle "awaiting moderation" or stock photo.
                if (car.thumbnailUrl) {
                    div.find("#imageLink").prop("href", car.imageUrl);
                    div.find("#image").prop("src", car.thumbnailUrl);
                } else {
                    div.find("#imageLink").remove();
                }

                return div;
            });

            // Build 3 columns of cards using Bootstrap columns.
            let i = 0;
            while (i < cards.length) {
                let row = $(`<div class="row"></div>`);
                let j = 0;
                while (j < 3 && i < cards.length) {
                    let col = $(`<div class="col-md-4"></div>`)
                    col.append(cards[i]);
                    row.append(col);
                    i++;
                    j++;
                }
                container.append(row);
            }
        }).catch(error => {
            alert("Failed to fetch cars: " + error);
        });
}


//////// Display Map Blocks ////////
function mapBlockOverlay(latitude, longitude, color = "#007BFF") {
    // Determine if the offset should be positive or negative so
    // that the magnitude of the sum is always greater (i.e.
    // farther away from zero. This works in conjunction with
    // truncating the car longitude and latitude, which always
    // rounds toward zero.
    let latOffset = 0.04999 * Math.sign(latitude);
    let longOffset = 0.04999 * Math.sign(longitude);
    return new mapkit.PolygonOverlay([
        new mapkit.Coordinate(latitude, longitude),
        new mapkit.Coordinate(latitude, longitude + longOffset),
        new mapkit.Coordinate(latitude + latOffset, longitude + longOffset),
        new mapkit.Coordinate(latitude + latOffset, longitude),
    ], {
        style: new mapkit.Style({
            fillColor: color,
            fillOpacity: 0.15,
            strokeColor: "#FF0000",
            strokeOpacity: 0.5,
            lineWidth: 1,
            lineJoin: "round",
            lineDash: [2, 2, 6, 2, 6, 2]
        }),
        visible: true,
        enabled: true,
    });
}

function buildOverlays(mapBlocks) {
    if (!mapBlocks) {
        return [];
    }
    return mapBlocks.map(block => {
        let overlay = mapBlockOverlay(block.latitude, block.longitude);
        overlay.data = { id: block.id };
        overlay.addEventListener("select", function (event) {
            displayCars(event.target.data.id);
        });
        overlay.addEventListener("deselect", function (event) {
            // Figure out how to cancel the previous request or
            // prevent the info from being displayed.
            // hideCars();
        });
        return overlay;
    });
}

// The maximum longitude or latitude span to display overlays.
const maxSpan = 2;
var overlays = [];
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
        .then(res => {
            handleErrors(res);
            return res.json();
        })
        .then(result => {
            map.removeOverlays(overlays);
            overlays = buildOverlays(result.mapBlocks);
            map.addOverlays(overlays);
        }).catch(error => {
            console.log("Failed to get map blocks: " + error);
        });
}

map.addEventListener("region-change-end", function (event) {
    fetchVisibleOverlays();

    if (addCarAnnotation && !isVisible(addCarAnnotation)) {
        addCarAnnotation.coordinate = map.center;
        drawAddCarOverlay(addCarAnnotation.coordinate);
    }
});
