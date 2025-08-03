function initDataConfirm() {
  document.body.addEventListener("click", (event) => {
    if (!event.target.closest(":is(a, button)[data-confirm]")) return;

    handleConfirmationMessage(event.target, (url, redirectURL) => {
      const request = new RequestBuilder(url);
      request.withRedirect("manual").withCallback((response) => {
        if (redirectURL) {
          window.location.href = redirectURL;
        } else if (response.type == "opaqueredirect" && response.url) {
          window.location.href = response.url;
        } else {
          window.location.reload();
        }
      });
      request.execute();
    });
  });
}

function initServiceWorker() {
  if ("serviceWorker" in navigator === false) return;

  const serviceWorkerURL = document.body.dataset.serviceWorkerUrl;
  if (!serviceWorkerURL) return;

	navigator.serviceWorker.
    register(ttpolicy.createScriptURL(serviceWorkerURL), {type: "module"}).
    catch((error) => {
      console.warn(`Service worker registration failed: ${error}`);
    });
}

function initCommentLinks() {
  document.body.addEventListener("click", (event) => {
    if (event.target.closest("a[data-comments-link=true]")) {
      handleEntryStatus("next", event.target, true);
    }
  });
}

initDataConfirm();
initServiceWorker();
initCommentLinks();
