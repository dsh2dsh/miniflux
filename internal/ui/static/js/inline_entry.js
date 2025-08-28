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

    body.addEventListener("htmx:sendError", (event) => {
      const el = event.detail.elt;
      if (el.closest(".item-title a")) {
        this.inlineFailed(el.closest(".item"), event.detail);
      };
    });

    body.addEventListener("htmx:responseError", (event) => {
      const el = event.detail.elt;
      if (el.closest(".item-title a")) {
        this.inlineFailed(el.closest(".item"), event.detail);
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
    const article = t.content.cloneNode(true);

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
    const loadingError = t.content.cloneNode(true);
    loadingError.querySelector(".errorText").innerText = detail.error;
    loadingError.querySelector(".responseText").innerText = detail.xhr.responseText;
    detail.target.replaceWith(loadingError);
    showToastNotification("error",
      `${detail.error}: ${detail.xhr.responseText}`);
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
    if (!this.setButtonLoading(button)) return;

    const item = button.closest(".item");
    item.addEventListener("htmx:afterSettle", (event) => {
      if (event.detail.elt.matches(".entry-content.download")) {
        button.parentElement.remove();
      }
    }, { once: true });

    button.addEventListener("htmx:sendError", (event) => {
      this.downloadFailed(button, event.detail);
    });

    button.addEventListener("htmx:responseError", (event) => {
      this.downloadFailed(button, event.detail);
    });
  }

  setButtonLoading(button) {
    if (button.querySelector(".htmx-indicator")) return false;

    const t = document.querySelector("template#entry-downloading");
    button.appendChild(t.content.cloneNode(true));
    return true;
  }

  downloadFailed(button, detail) {
    showToastNotification("error",
      `${detail.error}: ${detail.xhr.responseText}`);
  }

  downloaded(item) {
    item.classList.add("downloaded");
  }
}

const entryInliner = new InlineEntry();
