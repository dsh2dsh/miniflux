class InlineEntry {
  constructor() {
    const body = document.body;

    body.addEventListener("htmx:trigger", (event) => {
      const el = event.detail.elt;
      if (el.closest(".item-title a")) {
        this.beginInline(el);
      };
    });

    body.addEventListener("htmx:beforeRequest", (event) => {
      const target = event.detail.target;
      if (target.matches(".entry-content.inline")) {
        this.downloadingOriginal(event.target);
      }
    });

    body.addEventListener("htmx:afterSettle", (event) => {
      const el = event.detail.elt;
      if (el.matches(".item > .loaded")) {
        this.entryInlined(el.closest(".item"));
      } else if (el.matches(".entry-content.download")) {
        this.downloaded(el.closest(".item"));
      }
    });

    if (body.dataset.markAsReadOnView === "true") {
      body.addEventListener("click", (event) => {
        const el = event.target;
        if (el.closest(".item-title a")) {
          this.originalLinkClick(el.closest(".item"));
        };
      });
    };
  }

  beginInline(title) {
    this.addLoadingTarget(title.closest(".item"));
    title.addEventListener("htmx:confirm", (event) => {
      event.preventDefault();
      this.nextEventCycle(() => event.detail.issueRequest());
    });
  }

  addLoadingTarget(item) {
    const t = document.querySelector("template#entry-loading-inline");
    item.querySelector(".item-header").after(t.content.cloneNode(true));
  }

  nextEventCycle(fn) {
    setTimeout(fn, 0);
  }

  entryInlined(item) {
    const titleLink = item.querySelector(".item-title a");
    titleLink.setAttribute("hx-disable", "");
    htmx.process(titleLink);
    item.classList.add("with-inline-content");
  }

  originalLinkClick(item) {
    if (item.classList.contains("with-inline-content")) {
      markItemsRead([item]);
    };
  }

  downloadingOriginal(button) {
    this.setButtonLoading(button);
    const item = button.closest(".item");
    item.addEventListener("htmx:afterSettle", (event) => {
      if (event.detail.elt.matches(".entry-content.download")) {
        button.parentElement.remove();
      }
    }, { once: true });
  }

  setButtonLoading(button) {
    const originalLabel = button.querySelector(".icon-label")
    const loadingLabel = createIconLabelElement(
      document.body.dataset.labelLoading);
    button.replaceChild(loadingLabel, originalLabel);
    return originalLabel;
  }

  downloaded(item) {
    item.classList.add("downloaded");
  }
}

const entryInliner = new InlineEntry();
