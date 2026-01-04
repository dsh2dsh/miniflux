class SwitchTheme {
  constructor() {
    this.colorScheme = document.querySelector('meta[name="color-scheme"]');
    document.body.addEventListener("htmx:beforeSwap", event => {
      const el = event.target;
      if (el.matches("#header-menu [data-color-scheme]"))
        this.switchColorScheme(el.dataset.colorScheme);
    });
  }

  switchColorScheme(mode) {
    this.colorScheme.setAttribute(
      "content", mode == "system" ? "light dark" : mode);
    document.body.dataset.colorScheme = mode;
  }
}

const themeSwitcher = new SwitchTheme()
