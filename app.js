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
        color: "#c969e0"
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


var map = new mapkit.Map("map");
map.addEventListener("region-change-start", function (event) {
    console.log("Region Change Started");
});

fetch("/cars")
    .then(response => response.json())
    .then(result => {
        var annotations = result.cars.map(car => {
            return annotation({
                coordinate: new mapkit.Coordinate(car.longitude, car.latitude),
                title: car.year + " " + car.brand + " " + car.model + " " + car.trim,
                phone: car.id,
                url: ""
            });
        });
        map.showItems(annotations);
    });

// var MarkerAnnotation = mapkit.MarkerAnnotation
// var sfo = new mapkit.Coordinate(37.616934, -122.383790)

// var sfoRegion = new mapkit.CoordinateRegion(
//     new mapkit.Coordinate(37.616934, -122.383790),
//     new mapkit.CoordinateSpan(0.167647972, 0.354985255)
// );

// var sfoAnnotation = new MarkerAnnotation(sfo, { color: "#f4a56d", title: "SFO", glyphText: "✈️" });
// map.showItems([sfoAnnotation]);

// map.region = sfoRegion;

var searcher = new mapkit.Search({ region: map.region });

function submitSearch() {
    search($("#query").val());
}

function search(query) {
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
