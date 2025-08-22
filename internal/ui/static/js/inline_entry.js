class InlineEntry {
  constructor() {
    this.listenAfterSettle();
    this.listenLinkClick();
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

  downloaded(el) {
    const item = el.closest(".item")
    item.classList.add("downloaded");
  }
}

const entryInliner = new InlineEntry();
