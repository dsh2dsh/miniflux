class SwitchTheme {
  constructor() {
    this.colorScheme = document.querySelector('meta[name="color-scheme"]');

    document.querySelectorAll("#header-menu [data-color-scheme]")
      .forEach(el => {
        el.addEventListener("htmx:beforeSwap", event => {
          this.switchColorScheme(el.dataset.colorScheme);
        });
      });
  }

  switchColorScheme(mode) {
    this.colorScheme.setAttribute("content", mode);
    document.body.dataset.colorScheme = mode;
  }
}

const themeSwitcher = new SwitchTheme()
