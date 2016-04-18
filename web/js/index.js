// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.
/* jshint  strict: true */

Ractive.DEBUG = false;

(function() {
    "use strict";

    var r = new Ractive({
        el: "body",
        template: "#tMain",
        data: function() {
            return {
                project: null,
                version: null,
				stages: null,
                stage: null,
                projects: [],
                error: null,
            };
        },
    });

    setPaths();


    function setPaths() {
        var paths = window.location.pathname.split("/");

        if (paths.length <= 1) {
            getProjects();
            return;
        }
        if (!paths[1]) {
            getProjects();
            return;
        }

        if (paths[1] == "project") {
            if (paths[2]) {
                getProject(paths[2]);
                if (paths[3]) {
                    getVersion(paths[2], paths[3]);
                    if (paths[4]) {
                        getStage(paths[2], paths[3], paths[4]);
                    }
                }
            }
            getProjects();
            return;
        }

        r.set("error", "Path Not found!");
    }


    function getProjects() {
        get("/log/",
            function(result) {
                for (var i = 0; i < result.data.length; i++) {
                    setStatus(result.data[i]);
                }

                r.set("projects", result.data);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getProject(id) {
        get("/log/" + id,
            function(result) {
                r.set("project", result.data);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getVersion(id, version) {
        r.set("version", version);
        get("/log/" + id + "/" + version,
            function(result) {
                r.set("stages", result.data);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getStage(id, version, stage) {
        get("/log/" + id + "/" + version + "/" + stage,
            function(result) {
                r.set("stage", result.data);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function setStatus(project) {
        //statuses 
        if (project.lastLog.version.trim() == project.releaseVersion.trim()) {
            project.status = "Success";
        } else {
            if (project.lastLog.stage == "loading") {
                project.status = "Load Failing";
            } else if (project.lastLog.stage == "fetching") {
                project.status = "Fetch Failing";
            } else if (project.lastLog.stage == "building") {
                project.status = "Build Failing";
            } else if (project.lastLog.stage == "testing") {
                project.status = "Tests Failing";
            } else if (project.lastLog.stage == "releasing") {
                project.status = "Release Failing";
            } else {
                project.status = "Failing";
            }
        }
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

function err(response) {
    "use strict";
    var error = {
        message: "An error occurred and has been logged",
    };

    if (typeof response === "string") {
        error.message = response;
    }
    return error;
}
