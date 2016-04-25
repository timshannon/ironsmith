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
                currentStage: null,
                logs: null,
                projects: [],
                error: null,
                formatDate: formatDate,
                releases: {},
            };
        },
        decorators: {
            menu: function(node) {
                new PureDropdown(node);
                return {
                    teardown: function() {
                        return;
                    },
                };
            },
        },
    });

    setPaths();


    r.on({
        "triggerBuild": function(event) {
            event.original.preventDefault();
            var secret = window.prompt("Please enter the trigger secret for this project:");
            triggerBuild(r.get("project.id"), secret);
        },
    });


    function triggerBuild(projectID, secret) {
        ajax("POST", "/trigger/" + projectID, {
                secret: secret
            },
            function(result) {
                window.location = "/";
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }


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
                if (paths[3]) {
                    if (paths[4]) {
                        getStage(paths[2], paths[3], paths[4]);
                    }
                    getVersion(paths[2], paths[3]);
                }
                getProject(paths[2]);
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
                    hasRelease(result.data[i].id, "");
                }

                result.data.sort(function(a, b) {
                    if (a.name > b.name) {
                        return 1;
                    }
                    if (a.name < b.name) {
                        return -1;
                    }
                    return 0;
                });
                r.set("projects", result.data);

                window.setTimeout(getProjects, 10000);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getProject(id) {
        get("/log/" + id,
            function(result) {
                r.set("project", result.data);
                if (result.data.versions) {
                    for (var i = 0; i < result.data.versions.length; i++) {
                        hasRelease(result.data.id, result.data.versions[i].version);
                    }
                }
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getVersion(id, version) {
        get("/log/" + id + "/" + version,
            function(result) {
                if (!result.data || !result.data.length || !result.data[0].version) {
                    r.set("version", version);
                } else {
                    r.set("version", result.data[0].version);
                }
                r.set("stages", result.data);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function getStage(id, version, stage) {
        get("/log/" + id + "/" + version + "/" + stage,
            function(result) {
                r.set("logs", result.data);
                r.set("currentStage", stage);
            },
            function(result) {
                r.set("error", err(result).message);
            });
    }

    function hasRelease(id, version) {
        /*/release/<project-id>/<version>*/
        get("/release/" + id + "/" + version,
            function(result) {
                r.set("releases." + id + version, result.data);
            },
            function(result) {
                r.set("releases." + id + version, undefined);
            });


    }

    function setStatus(project) {
        //statuses 
        if (project.stage != "waiting") {
            project.status = project.stage;
        } else if (!project.lastLog || !project.lastLog.version) {
            project.status = "waiting";
        } else if (project.lastLog.version.trim() == project.releaseVersion.trim()) {
            project.status = "Successfully Released";
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
                    var result;
                    try {
                        result = JSON.parse(req.responseText);
                    } catch (e) {
                        result = "";
                    }
                    success(result);
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

    var sendData;
    if (type != "get") {
        req.setRequestHeader("Content-Type", "application/json");
        sendData = JSON.stringify(data);
    }

    req.send(sendData);
}

function get(url, success, error) {
    "use strict";
    ajax("GET", url, null, success, error);
}

function err(response) {
    "use strict";
    var error = {
        message: "An error occurred",
    };

    if (typeof response === "string") {
        error.message = response;
    } else {
        error.message = JSON.parse(response.responseText).message;
    }

    return error;
}

function formatDate(strDate) {
    "use strict";
    var date = new Date(strDate);
    if (!date) {
        return "";
    }
    return date.toLocaleDateString() + " at " + date.toLocaleTimeString();
}


function PureDropdown(dropdownParent) {
    "use strict";

    var PREFIX = 'pure-',
        ACTIVE_CLASS_NAME = PREFIX + 'menu-active',
        ARIA_ROLE = 'role',
        ARIA_HIDDEN = 'aria-hidden',
        MENU_OPEN = 0,
        MENU_CLOSED = 1,
        MENU_PARENT_CLASS_NAME = 'pure-menu-has-children',
        MENU_ACTIVE_SELECTOR = '.pure-menu-active',
        MENU_LINK_SELECTOR = '.pure-menu-link',
        MENU_SELECTOR = '.pure-menu-children',
        DISMISS_EVENT = (window.hasOwnProperty &&
            window.hasOwnProperty('ontouchstart')) ?
        'touchstart' : 'mousedown',

        ARROW_KEYS_ENABLED = true,

        ddm = this; // drop down menu

    this._state = MENU_CLOSED;

    this.show = function() {
        if (this._state !== MENU_OPEN) {
            this._dropdownParent.classList.add(ACTIVE_CLASS_NAME);
            this._menu.setAttribute(ARIA_HIDDEN, false);
            this._state = MENU_OPEN;
        }
    };

    this.hide = function() {
        if (this._state !== MENU_CLOSED) {
            this._dropdownParent.classList.remove(ACTIVE_CLASS_NAME);
            this._menu.setAttribute(ARIA_HIDDEN, true);
            this._link.focus();
            this._state = MENU_CLOSED;
        }
    };

    this.toggle = function() {
        this[this._state === MENU_CLOSED ? 'show' : 'hide']();
    };

    this.halt = function(e) {
        e.stopPropagation();
        e.preventDefault();
    };

    this._dropdownParent = dropdownParent;
    this._link = this._dropdownParent.querySelector(MENU_LINK_SELECTOR);
    this._menu = this._dropdownParent.querySelector(MENU_SELECTOR);
    this._firstMenuLink = this._menu.querySelector(MENU_LINK_SELECTOR);

    // Set ARIA attributes
    this._link.setAttribute('aria-haspopup', 'true');
    this._menu.setAttribute(ARIA_ROLE, 'menu');
    this._menu.setAttribute('aria-labelledby', this._link.getAttribute('id'));
    this._menu.setAttribute('aria-hidden', 'true');
    [].forEach.call(
        this._menu.querySelectorAll('li'),
        function(el) {
            el.setAttribute(ARIA_ROLE, 'presentation');
        }
    );
    [].forEach.call(
        this._menu.querySelectorAll('a'),
        function(el) {
            el.setAttribute(ARIA_ROLE, 'menuitem');
        }
    );

    // Toggle on click
    this._link.addEventListener('click', function(e) {
        e.stopPropagation();
        e.preventDefault();
        ddm.toggle();
    });

    // Keyboard navigation
    document.addEventListener('keydown', function(e) {
        var currentLink,
            previousSibling,
            nextSibling,
            previousLink,
            nextLink;

        // if the menu isn't active, ignore
        if (ddm._state !== MENU_OPEN) {
            return;
        }

        // if the menu is the parent of an open, active submenu, ignore
        if (ddm._menu.querySelector(MENU_ACTIVE_SELECTOR)) {
            return;
        }

        currentLink = ddm._menu.querySelector(':focus');

        // Dismiss an open menu on ESC
        if (e.keyCode === 27) {
            /* Esc */
            ddm.halt(e);
            ddm.hide();
        }
        // Go to the next link on down arrow
        else if (ARROW_KEYS_ENABLED && e.keyCode === 40) {
            /* Down arrow */
            ddm.halt(e);
            // get the nextSibling (an LI) of the current link's LI
            nextSibling = (currentLink) ? currentLink.parentNode.nextSibling : null;
            // if the nextSibling is a text node (not an element), go to the next one
            while (nextSibling && nextSibling.nodeType !== 1) {
                nextSibling = nextSibling.nextSibling;
            }
            nextLink = (nextSibling) ? nextSibling.querySelector('.pure-menu-link') : null;
            // if there is no currently focused link, focus the first one
            if (!currentLink) {
                ddm._menu.querySelector('.pure-menu-link').focus();
            } else if (nextLink) {
                nextLink.focus();
            }
        }
        // Go to the previous link on up arrow
        else if (ARROW_KEYS_ENABLED && e.keyCode === 38) {
            /* Up arrow */
            ddm.halt(e);
            // get the currently focused link
            previousSibling = (currentLink) ? currentLink.parentNode.previousSibling : null;
            while (previousSibling && previousSibling.nodeType !== 1) {
                previousSibling = previousSibling.previousSibling;
            }
            previousLink = (previousSibling) ? previousSibling.querySelector('.pure-menu-link') : null;
            // if there is no currently focused link, focus the last link
            if (!currentLink) {
                ddm._menu.querySelector('.pure-menu-item:last-child .pure-menu-link').focus();
            }
            // else if there is a previous item, go to the previous item
            else if (previousLink) {
                previousLink.focus();
            }
        }
    });

    // Dismiss an open menu on outside event
    document.addEventListener(DISMISS_EVENT, function(e) {
        var target = e.target;
        if (target !== ddm._link && !ddm._menu.contains(target)) {
            ddm.hide();
            ddm._link.blur();
        }
    });

}
