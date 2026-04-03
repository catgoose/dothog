/**
 * Interval control helpers called from hyperscript.
 * Each interval slider wrapper <div> stores state in data-* attributes
 * and a _ivUnit expando property for the current unit index.
 * Self-initializing: _ivUnit is set lazily on first use.
 */
(function () {
  var units = ['ms', 's', 'min', 'h'];
  var configs = {
    ms:  { min: 100, max: 2000, step: 100, mult: 1 },
    s:   { min: 1,   max: 60,   step: 1,   mult: 1000 },
    min: { min: 1,   max: 60,   step: 1,   mult: 60000 },
    h:   { min: 1,   max: 24,   step: 1,   mult: 3600000 }
  };

  function ensureInit(el) {
    if (el._ivUnit == null) {
      var unit = el.dataset.unit || 's';
      var idx = units.indexOf(unit);
      el._ivUnit = idx >= 0 ? idx : 1;
    }
  }

  window._ivUp = function (el) { _ivCycle(el, 1); };
  window._ivDown = function (el) { _ivCycle(el, -1); };

  /** Cycle unit. dir=1 forward (ms→s→min→h), dir=-1 backward. */
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
    if (val > cfg.max) val = cfg.min;
    if (val < cfg.min) val = cfg.min;

    input.min = cfg.min;
    input.max = cfg.max;
    input.step = cfg.step;
    input.value = val;
    if (display) display.textContent = val;
    if (btn) btn.textContent = unit;

    _ivPost(el);
  };

  /** POST the current interval in ms. Skips if the range input is disabled. */
  window._ivPost = function (el) {
    if (!el) return;
    ensureInit(el);
    var input = el.querySelector('input[type=range]');
    if (!input) return;
    // If inside a master control, only POST when the toggle is checked
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
