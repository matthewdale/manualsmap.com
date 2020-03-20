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

var newCarAnnotation = null;
function addCar() {
    if (newCarAnnotation) {
        map.removeAnnotation(newCarAnnotation);
    }
    let form = $("#addcarform").clone();
    form.on('submit', function () {
        // TODO: Form validation.
        // TODO: Submit data.
        console.log($(this));
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

// The maximum longitude or latitude span to display overlays/
const maxSpan = 0.6;
var overlays = [];
map.addEventListener("region-change-end", function (event) {
    console.log("region-change-end");
    console.log(map.region.toBoundingRegion());

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

            let style = new mapkit.Style({
                strokeColor: "#F00",
                strokeOpacity: .2,
                lineWidth: 2,
                lineJoin: "round",
                lineDash: [2, 2, 6, 2, 6, 2]
            });

            overlays = result.map_blocks.map(block => {
                // Determine if the offset should be positive or negative so
                // that the magnitude of the sum is always greater (i.e.
                // farther away from zero. This works in conjunction with
                // truncating the car longitude and latitude, which always
                // rounds toward zero.
                let latOffset = 0.01 * Math.sign(block.latitude);
                let longOffset = 0.01 * Math.sign(block.longitude);
                let overlay = new mapkit.PolygonOverlay([
                    new mapkit.Coordinate(block.latitude, block.longitude),
                    new mapkit.Coordinate(block.latitude, block.longitude + longOffset),
                    new mapkit.Coordinate(block.latitude + latOffset, block.longitude + longOffset),
                    new mapkit.Coordinate(block.latitude + latOffset, block.longitude),
                ], {
                    style: style,
                    visible: true,
                    enabled: true,
                    data: { id: block.id },
                });
                overlay.addEventListener("select", function (event) {
                    fetch(`/mapblocks/${event.target.data.id}/cars`)
                        .then(response => response.json())
                        .then(result => {
                            $("#carsInfo").text(JSON.stringify(result.cars));
                        });
                });
                overlay.addEventListener("deselect", function (event) {
                    // Figure out how to cancel the previous request or
                    // prevent the info from being displayed.
                    $("#carsInfo").text("");
                });
                return overlay;
            });
            map.addOverlays(overlays);
        });
})