// setup:feature:demo
/**
 * Interval control helpers called from hyperscript.
 *
 * Each interval slider wrapper (.iv-wrap) stores configuration in data-*
 * attributes and a `_ivUnit` expando for the current unit index.
 * The unit index is lazily initialized from `data-unit` on first interaction.
 *
 * @module interval-control
 */
(function () {
  /** @type {string[]} Ordered time units for cycling. */
  var units = ['ms', 's', 'min', 'h'];

  /**
   * Per-unit slider configuration.
   *
   * Ranges are intentionally non-overlapping so each millisecond value maps
   * to exactly one natural unit:
   *   ms:  100 – 900   (step 100)
   *   s:   1 – 59      (= 1 000 – 59 000 ms)
   *   min: 1 – 59      (= 60 000 – 3 540 000 ms)
   *   h:   1 – 24      (= 3 600 000 – 86 400 000 ms)
   *
   * @type {Object<string, {min: number, max: number, step: number, mult: number}>}
   */
  var configs = {
    ms:  { min: 100, max: 900,  step: 100, mult: 1 },
    s:   { min: 1,   max: 59,   step: 1,   mult: 1000 },
    min: { min: 1,   max: 59,   step: 1,   mult: 60000 },
    h:   { min: 1,   max: 24,   step: 1,   mult: 3600000 }
  };

  /**
   * Lazily initialize the unit index from the element's data-unit attribute.
   * @param {HTMLElement} el - The .iv-wrap container element.
   */
  function ensureInit(el) {
    if (el._ivUnit == null) {
      var unit = el.dataset.unit || 's';
      var idx = units.indexOf(unit);
      el._ivUnit = idx >= 0 ? idx : 1;
    }
  }

  /**
   * Cycle unit forward (ms → s → min → h).
   * @param {HTMLElement} el - The .iv-wrap container element.
   */
  window._ivUp = function (el) { _ivCycle(el, 1); };

  /**
   * Cycle unit backward (h → min → s → ms).
   * @param {HTMLElement} el - The .iv-wrap container element.
   */
  window._ivDown = function (el) { _ivCycle(el, -1); };

  /**
   * Cycle the slider's unit and reconfigure its range.
   *
   * When switching units, the current millisecond value is converted to the
   * new unit and clamped to the new range.  If the converted value would be
   * below the minimum (e.g. 500 ms → 0 s), the slider snaps to the new
   * unit's minimum instead of showing zero.
   *
   * @param {HTMLElement} el  - The .iv-wrap container element.
   * @param {number}      dir - 1 = forward, -1 = backward.
   */
  function _ivCycle(el, dir) {
    if (!el) return;
    ensureInit(el);
    var input = el.querySelector('input[type=range]');
    var display = el.querySelector('.iv-display');
    var btn = el.querySelector('button');
    if (!input) return;

    var oldCfg = configs[units[el._ivUnit]];
    var ms = parseInt(input.value) * oldCfg.mult;

    el._ivUnit = (el._ivUnit + (dir || 1) + units.length) % units.length;
    var unit = units[el._ivUnit];
    var cfg = configs[unit];

    var val = Math.round(ms / cfg.mult);
    if (val < cfg.min) val = cfg.min;
    if (val > cfg.max) val = cfg.max;

    input.min = cfg.min;
    input.max = cfg.max;
    input.step = cfg.step;
    input.value = val;
    if (display) display.textContent = val;
    if (btn) btn.textContent = unit;

    _ivPost(el);
  }

  /**
   * POST the current interval (in milliseconds) and unit to the server.
   *
   * Reads the slider value and current unit, converts to ms, and sends a
   * fetch POST to the URL in data-post-url.  Skips the POST when the
   * element is inside a master control whose toggle is unchecked.
   *
   * @param {HTMLElement} el - The .iv-wrap container element.
   */
  window._ivPost = function (el) {
    if (!el) return;
    ensureInit(el);
    var input = el.querySelector('input[type=range]');
    if (!input) return;
    // If inside a master control, only POST when the toggle is checked.
    var master = el.closest('.master-slider');
    if (master) {
      var toggle = master.parentElement.querySelector('.master-toggle');
      if (toggle && !toggle.checked) return;
    }
    var cfg = configs[units[el._ivUnit]];
    var ms = parseInt(input.value) * cfg.mult;

    var url = el.dataset.postUrl;
    var key = el.dataset.targetKey;
    var value = el.dataset.targetValue;
    if (!url) return;

    var params = new URLSearchParams();
    params.set(key, value);
    params.set('interval_ms', ms.toString());
    params.set('unit', units[el._ivUnit]);
    /** @type {HTMLMetaElement|null} */
    var t = document.querySelector('meta[name="csrf-token"]');
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'X-CSRF-Token': t ? t.content : ''
      },
      body: params.toString()
    });
  };
})();
