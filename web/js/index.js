// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.
/* jshint  strict: true */

(function() {
    "use strict";

    var r = new Ractive({
        el: "body",
        template: "#tMain",
        data: function() {
            return {
                project: null,
                version: null,
                stage: null,
                projects: [],
            };
        },
    });

    getProjects();


    function getProjects() {
        get("/log/",
            function(result) {
                r.set("projects", result.data);
            },
            function(result) {
                console.log("error", result);
            });
    }

})();

function ajax(type, url, data, success, error) {
    "use strict";
    var req = new XMLHttpRequest();
    req.open(type, url);

    if (success || error) {
        req.onload = function() {
            if (req.status >= 200 && req.status < 400) {
                if (success && typeof success === 'function') {
                    success(JSON.parse(req.responseText));
                }
                return;
            }

            //failed
            if (error && typeof error === 'function') {
                error(req);
            }
        };
        req.onerror = function() {
            if (error && typeof error === 'function') {
                error(req);
            }
        };
    }

    if (type != "get") {
        req.setRequestHeader("Content-Type", "application/json");
    }

    req.send(data);
}

function get(url, success, error) {
    "use strict";
    ajax("GET", url, null, success, error);
}

