// Landmark annotation custom callout
function calloutForLandmarkAnnotation(annotation) {
    var div = document.createElement("div");
    div.className = "landmark";

    var title = div.appendChild(document.createElement("h1"));
    title.textContent = annotation.landmark.title;

    var section = div.appendChild(document.createElement("section"));

    var phone = section.appendChild(document.createElement("p"));
    phone.className = "phone";
    phone.textContent = annotation.landmark.phone;

    var link = section.appendChild(document.createElement("p"));
    link.className = "homepage";
    var a = link.appendChild(document.createElement("a"));
    a.href = annotation.landmark.url;
    a.textContent = "website";

    return div;
}

function annotation(landmark) {
    var annotation = new mapkit.MarkerAnnotation(landmark.coordinate, {
        callout: landmarkAnnotationCallout,
        color: "#c969e0",
        draggable: true,
    });
    annotation.landmark = landmark;
    return annotation;
}

// Landmark annotation callout delegate
var CALLOUT_OFFSET = new DOMPoint(-148, -78);
var landmarkAnnotationCallout = {
    calloutElementForAnnotation: function (annotation) {
        return calloutForLandmarkAnnotation(annotation);
    },

    calloutAnchorOffsetForAnnotation: function (annotation, element) {
        return CALLOUT_OFFSET;
    },

    calloutAppearanceAnimationForAnnotation: function (annotation) {
        return "scale-and-fadein .4s 0 1 normal cubic-bezier(0.4, 0, 0, 1.5)";
    }
};


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

var style = new mapkit.Style({
    strokeColor: "#F00",
    strokeOpacity: .2,
    lineWidth: 2,
    lineJoin: "round",
    lineDash: [2, 2, 6, 2, 6, 2]
});

var rectangle = new mapkit.PolygonOverlay([
    new mapkit.Coordinate(47.63, -122.36), // top left
    new mapkit.Coordinate(47.63, -122.35), // top right
    new mapkit.Coordinate(47.62, -122.35), // bottom right
    new mapkit.Coordinate(47.62, -122.36), // bottom left
], {
    style: style,
    visible: true,
    enabled: true,
    // selected: true,
    data: { blah: "square!" },
});
rectangle.addEventListener("select", function (event) {
    $("#carsInfo").text(JSON.stringify(event.target.data));
});
rectangle.addEventListener("deselect", function (event) {
    $("#carsInfo").text("");
});
map.addOverlay(rectangle);

var rectangle = new mapkit.PolygonOverlay([
    new mapkit.Coordinate(47.63, -122.35), // top left
    new mapkit.Coordinate(47.63, -122.34), // top right
    new mapkit.Coordinate(47.62, -122.34), // bottom right
    new mapkit.Coordinate(47.62, -122.35), // bottom left
], {
    style: style,
    visible: true,
    enabled: true,
    // selected: true,
    data: { blah: "square!" },
});
rectangle.addEventListener("select", function (event) {
    $("#carsInfo").text(JSON.stringify(event.target.data));
});
rectangle.addEventListener("deselect", function (event) {
    $("#carsInfo").text("");
});
map.addOverlay(rectangle);

fetch("/cars")
    .then(response => response.json())
    .then(result => {
        var annotations = result.cars.map(car => {
            return annotation({
                coordinate: new mapkit.Coordinate(car.latitude, car.longitude),
                title: car.year + " " + car.brand + " " + car.model + " " + car.trim,
                phone: car.id,
                url: ""
            });
        });
        map.showItems(annotations);
    });

function submitSearch() {
    search($("#searchQuery").val());
}

function search(query) {
    var searcher = new mapkit.Search({ region: map.region });
    searcher.search(query, function (error, data) {
        if (error) {
            // Handle search error
            return;
        }
        var annotations = data.places.map(function (place) {
            var annotation = new mapkit.MarkerAnnotation(place.coordinate);
            annotation.title = place.name;
            annotation.subtitle = place.formattedAddress;
            annotation.color = "#9B6134";
            return annotation;
        });
        map.showItems(annotations);
    });
}

// map.addEventListener("select", function (event) {
//     console.log("select");
// });
// map.addEventListener("deselect", function (event) {
//     console.log("deselect");
// });
// map.addEventListener("drag-start", function (event) {
//     console.log("drag-start");
// });
// map.addEventListener("dragging", function (event) {
//     console.log("dragging");
// });
map.addEventListener("drag-end", function (event) {
    console.log("drag-end");
    var geocoder = new mapkit.Geocoder({
        language: "en-GB",
    });
    console.log(event.annotation);
    geocoder.reverseLookup(event.annotation.coordinate, function (error, data) {
        if (error) {
            // Handle reverse lookup error.
            return;
        }
        var addresses = data.results.map(place => place.formattedAddress);
        console.log(addresses);
    });
});
// map.addEventListener("user-location-change", function (event) {
//     console.log("user-location-change");
// });
// map.addEventListener("user-location-error", function (event) {
//     console.log("user-location-error");
// });

// map.element.addEventListener("click", function (event) {
//     if (event.target.parentNode !== map.element) {
//         // This condition prevents clicks on controls. Binding to a 
//         // secondary click is another option to prevent conflict
//         return;
//     }
//     console.log("click on map");
//     console.log(event);
// });
