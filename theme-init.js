// Runs in <head> BEFORE first paint (kept external so it passes the strict CSP).
// Sets the saved light/dark theme AND the page background synchronously, so a full
// page load (e.g. clicking Home) never flashes the default white before the CSS
// bundle loads / the app renders.
try {
  var t = localStorage.getItem('atlas-theme');
  var theme = (t === 'dark' || t === 'light') ? t : 'light';
  var el = document.documentElement;
  el.setAttribute('data-theme', theme);
  el.style.backgroundColor = theme === 'dark' ? '#0F0F11' : '#F2F2F7';
} catch (e) { /* ignore */ }
