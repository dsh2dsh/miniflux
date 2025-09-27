class SwitchTheme {
  constructor() {
    document.querySelectorAll("#header-menu span[data-theme-switch]")
      .forEach((el) => {
        el.addEventListener("click", (event) => {
          event.preventDefault();
          const route = el.dataset.themeSwitch;
          this.switchTheme(route);
        });
      });
  }

  async switchTheme(route) {
    const resp = await fetch(route, {
      method: "POST",
      headers: {
        "Accept": "application/json",
      },
    });
    if (!resp.ok)
      throw new Error(`Response status: ${resp.status}`);

    const result = await resp.json();
    location.replace(location.href);
  }
}

const themeSwitcher = new SwitchTheme();
