class InlineEntry {
  static itemTitleLink = ".entry-item .item-title";

  constructor() {
    const body = document.body;

    body.addEventListener("click", (event) => {
      const el = event.target;
      if (el.closest(InlineEntry.itemTitleLink) && el.tagName === "A") {
        if (el.getAttribute("hx-ignore") === null)
          this.beginInline(el);
        else if (body.dataset.markAsReadOnView === "true")
          this.originalLinkClick(el.closest(".entry-item"));
      }
    }, true);

    body.addEventListener("htmx:before:request", (event) => {
      const target = event.detail.ctx.target;
      if (target.matches(".entry-content.inline"))
        this.downloadingOriginal(event.target);
    }, true);

    body.addEventListener("htmx:error", (event) => {
      const el = event.detail.ctx.sourceElement;
      if (el.closest(InlineEntry.itemTitleLink))
        this.inlineFailed(el.closest(".item"), event.detail);
    }, true);

    body.addEventListener("htmx:response:error", (event) => {
      const el = event.detail.ctx.sourceElement;
      if (el.closest(InlineEntry.itemTitleLink))
        this.inlineFailed(el.closest(".item"), event.detail);
    }, true);

    body.addEventListener("htmx:after:settle", (event) => {
      const el = event.target;
      if (el.matches(".entry-item > .loaded"))
        this.entryInlined(el.closest(".item"));
      else if (el.matches(".entry-content.download"))
        this.downloaded(el.closest(".item"));
    }, true);
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
    const article = document.importNode(t.content, true);

    const withError = item.querySelector(".entry-content.with-error");
    if (withError) {
      withError.replaceWith(article);
    } else {
      item.querySelector(".item-header").after(article);
    };
  }

  nextEventCycle(fn) {
    setTimeout(fn, 0);
  }

  inlineFailed(item, detail) {
    const t = document.querySelector("template#entry-loading-error");
    const loadingError = document.importNode(t.content, true);
    if (detail.error)
      loadingError.querySelector(".errorText").innerText = detail.error;
    else if (detail.response) {
      const ctx = event.detail.ctx;
      loadingError.querySelector(".errorText").innerText =
        `Unexpected server response: ${ctx.status} ${ctx.raw.statusText}`;
    }
    detail.target.replaceWith(loadingError);
  }

  entryInlined(item) {
    const titleLink = item.querySelector(".item-title > [hx-trigger]");
    titleLink.setAttribute("hx-ignore", "");
    htmx.process(titleLink, true);
    item.classList.add("with-inline-content");
  }

  originalLinkClick(item) {
    if (item.classList.contains("with-inline-content")) {
      markItemsRead([item]);
    };
  }

  downloadingOriginal(button) {
    if (!this.setButtonLoading(button)) return;

    const item = button.closest(".item");
    item.addEventListener("htmx:after:settle", (event) => {
      const el = event.target;
      if (el.matches(".entry-content.download")) {
        button.parentElement.remove();
        item.scrollIntoView();
      }
    }, { once: true });
  }

  setButtonLoading(button) {
    if (button.querySelector(".htmx-indicator")) return false;

    const t = document.querySelector("template#entry-downloading");
    button.appendChild(document.importNode(t.content, true));
    return true;
  }

  downloaded(item) {
    item.classList.add("downloaded");
  }
}

const entryInliner = new InlineEntry();
