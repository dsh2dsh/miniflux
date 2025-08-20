class InlineEntry {
  constructor() {
    this.listenEntryInlined();
    this.listenLinkClick();
  }

  listenEntryInlined() {
    document.body.addEventListener("htmx:afterSettle", (event) => {
      if (!event.target.matches(".item-header")) return;
      this.entryInlined(event.target);
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
}

const entryInliner = new InlineEntry();
