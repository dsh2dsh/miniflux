class InlineEntry {
  constructor() {
    this.listenBeforeSend();
    this.listenAfterSettle();
    this.listenLinkClick();
  }

  listenBeforeSend() {
    document.body.addEventListener("htmx:beforeSend", (event) => {
      const target = event.detail.target;
      if (target.matches(".item-header")) {
        this.loadingInline(target);
      } else if (target.matches(".entry-content.inline")) {
        this.downloading(event.target, target);
      }
    });
  }

  listenAfterSettle() {
    document.body.addEventListener("htmx:afterSettle", (event) => {
      const detail = event.detail;
      if (detail.target.matches(".item-header")) {
        this.entryInlined(detail.target);
      } else if (detail.elt.matches(".entry-content.download")) {
        this.downloaded(detail.elt);
      }
    });
  }

  listenLinkClick() {
    document.body.addEventListener("click", (event) => {
      if (!event.target.closest(".item-title a")) return;

      const item = event.target.closest(".item")
      if (!item || !item.classList.contains("with-inline-content")) return;
      if (document.body.dataset.markAsReadOnView === "true") {
        this.markItemRead(item);
      };
    });
  }

  loadingInline(el) {
    const t = document.querySelector("template#loading-indicator");
    el.after(t.content.cloneNode(true));
    el.addEventListener("htmx:beforeSwap", () => {
      const item = el.parentElement;
      item.querySelector(".htmx-indicator").remove();
    }, { once: true });
  }

  entryInlined(el) {
    const titleLink = el.querySelector(".item-title a");
    titleLink.dataset.hxDisable = "true";
    htmx.process(el);

    const item = el.closest(".item")
    item.classList.add("with-inline-content");
  }

  markItemRead(el) {
    markItemsRead([el]);
  }

  downloading(button, target) {
    this.setButtonLoading(button);
    const item = target.closest(".item");
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

  downloaded(el) {
    const item = el.closest(".item")
    item.classList.add("downloaded");
  }
}

const entryInliner = new InlineEntry();
