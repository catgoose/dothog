/**
 * tavern-container.js — Local polyfill for declarative delegated commands.
 *
 * Aligns with the upstream tavern-js PR #29 attribute model so this can be
 * replaced by the real thing once it ships:
 *
 *   tavern-command-delegate="<event>"   on the stable container
 *   tavern-command-target="<selector>"  CSS selector for closest()
 *   command-url="<url>"                 on actionable descendants
 *   command-*="<value>"                 collected into the JSON body
 *
 * Extensions beyond upstream (kept minimal):
 *   - Multiple delegate bindings via comma-separated event types
 *     e.g. tavern-command-delegate="pointerdown,click"
 *     First matching event wins; subsequent events for the same logical
 *     action are suppressed via a short dedup window.
 *   - tavern-command-optimistic="<name>" for named client-side effects
 *     dispatched as a "tavern:optimistic" CustomEvent on the target element.
 *   - Debug logging when the container has tavern-debug.
 *
 * Requires: Tavern.command() from tavern.min.js (loaded first).
 */
(function () {
  "use strict";

  var DEDUP_MS = 400;

  /**
   * Collect all command-* attributes (except command-url) into an object.
   */
  function collectBody(el) {
    var body = {};
    var attrs = el.attributes;
    for (var i = 0; i < attrs.length; i++) {
      var name = attrs[i].name;
      if (name === "command-url") continue;
      if (name.indexOf("command-") === 0) {
        body[name.slice(8)] = attrs[i].value;
      }
    }
    return body;
  }

  /**
   * Expand {token} placeholders in a URL from the element's attributes.
   * Looks for data-<token> first, then command-<token>, then plain <token>.
   */
  function expandURL(url, el) {
    return url.replace(/\{([^}]+)\}/g, function (_, token) {
      var camel = token.replace(/-([a-z])/g, function (_, c) {
        return c.toUpperCase();
      });
      // data-* attributes are accessible via dataset
      if (el.dataset && el.dataset[camel] !== undefined) {
        return encodeURIComponent(el.dataset[camel]);
      }
      // Try command-<token>
      var cmdVal = el.getAttribute("command-" + token);
      if (cmdVal !== null) return encodeURIComponent(cmdVal);
      // Try raw attribute
      var rawVal = el.getAttribute(token);
      if (rawVal !== null) return encodeURIComponent(rawVal);
      return "{" + token + "}";
    });
  }

  /**
   * Bind one event listener on the container for a given event type.
   */
  function bindEvent(container, eventType, selector, debug) {
    container.addEventListener(
      eventType,
      function (e) {
        var target = e.target.closest(selector);
        if (!target || !container.contains(target)) return;

        var url = target.getAttribute("command-url");
        if (!url) return;

        // Dedup: skip if this element fired recently (e.g. pointerdown then click).
        var now = Date.now();
        var last = target._tavernLastCmd || 0;
        if (now - last < DEDUP_MS) {
          if (debug)
            console.debug(
              "[tavern-container] dedup skip",
              eventType,
              url
            );
          return;
        }
        target._tavernLastCmd = now;

        e.preventDefault();
        e.stopPropagation();

        var resolvedURL = expandURL(url, target);
        var body = collectBody(target);

        if (debug)
          console.debug(
            "[tavern-container]",
            eventType,
            resolvedURL,
            body
          );

        // Optimistic behavior hook.
        var optimistic = target.getAttribute("tavern-command-optimistic");
        if (optimistic) {
          target.dispatchEvent(
            new CustomEvent("tavern:optimistic", {
              bubbles: true,
              detail: { name: optimistic, url: resolvedURL, target: target },
            })
          );
        }

        // Dispatch sent event (upstream compat).
        target.dispatchEvent(
          new CustomEvent("tavern:command-sent", {
            bubbles: true,
            detail: { url: resolvedURL, body: body },
          })
        );

        // Use Tavern.command if available, fall back to raw fetch.
        var p =
          typeof Tavern !== "undefined" && typeof Tavern.command === "function"
            ? Tavern.command(resolvedURL, body)
            : fetch(resolvedURL, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
              });

        p.then(
          function (response) {
            target.dispatchEvent(
              new CustomEvent("tavern:command-success", {
                bubbles: true,
                detail: {
                  url: resolvedURL,
                  body: body,
                  response: response,
                },
              })
            );
          },
          function (error) {
            target.dispatchEvent(
              new CustomEvent("tavern:command-error", {
                bubbles: true,
                detail: { url: resolvedURL, body: body, error: error },
              })
            );
          }
        );
      },
      // Use capture phase for pointerdown to beat any default handlers.
      eventType === "pointerdown"
    );
  }

  /**
   * Initialize all containers on the page.
   */
  function init() {
    var containers = document.querySelectorAll("[tavern-command-delegate]");
    containers.forEach(function (container) {
      var events = container
        .getAttribute("tavern-command-delegate")
        .split(",")
        .map(function (s) {
          return s.trim();
        })
        .filter(Boolean);
      var selector =
        container.getAttribute("tavern-command-target") || "[command-url]";
      var debug = container.hasAttribute("tavern-debug");

      if (debug)
        console.debug(
          "[tavern-container] init",
          container.id || container.tagName,
          events,
          selector
        );

      events.forEach(function (evt) {
        bindEvent(container, evt, selector, debug);
      });

      container._tavernContainerBound = true;
    });
  }

  // Auto-initialize.
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  // Re-scan after htmx settles in case new containers appear.
  document.addEventListener("htmx:afterSettle", function (e) {
    var container = e.target.closest("[tavern-command-delegate]");
    if (container && !container._tavernContainerBound) {
      init();
    }
  });
})();
