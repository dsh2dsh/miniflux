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
      const title = event.target.closest(".item-title a");
      if (!title || !title.dataset.hxDisable) return;
      if (document.body.dataset.markAsReadOnView === "true") {
        const item = event.target.closest(".item")
        this.markItemRead(item);
      };
    });
  }

  entryInlined(el) {
    const titleLink = el.querySelector(".item-title a");
    titleLink.dataset.hxDisable = "true";
    htmx.process(el);
  }

  markItemRead(el) {
    markItemsRead([el]);
  }
}

const entryInliner = new InlineEntry();
