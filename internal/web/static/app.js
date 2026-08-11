window.snackbarQueue = function () {
  return {
    messages: [],
    push(detail) {
      const message = {
        id: `${Date.now()}-${Math.random()}`,
        type: detail.type || "error",
        title: detail.title || "Request failed",
        message: detail.message || "The request could not be completed.",
        visible: true,
      };
      this.messages.push(message);
      window.setTimeout(() => this.dismiss(message), 3000);
    },
    dismiss(message) {
      message.visible = false;
      window.setTimeout(() => {
        this.messages = this.messages.filter((item) => item.id !== message.id);
      }, 700);
    },
    tone(type) {
      return {
        error: "status-overdue",
        warning: "status-near-due",
        success: "status-payment-received",
        info: "status-pending",
      }[type] || "status-overdue";
    },
  };
};

window.dispatchRequestError = function (xhr) {
  const status = xhr && xhr.status ? ` (${xhr.status})` : "";
  window.dispatchEvent(new CustomEvent("snackbar", {
    detail: {
      type: "error",
      title: "Request failed",
      message: xhr?.getResponseHeader("X-Feedback-Message") || `The request could not be completed${status}.`,
    },
  }));
};

window.multiSelect = function () {
  return {
    open: false,
    options: [],
    selected: [],
    initialize(fallbackInputs) {
      this.options = Array.from(this.$root.querySelectorAll("[data-multi-select-option]")).map((option) => ({
        value: option.dataset.value,
        label: option.dataset.label,
      }));
      const validValues = new Set(this.options.map((option) => option.value));
      this.selected = Array.from(fallbackInputs.querySelectorAll("input"))
        .map((input) => input.value)
        .filter((value, index, values) => validValues.has(value) && values.indexOf(value) === index);
      fallbackInputs.remove();
    },
    toggle(value) {
      if (this.selected.includes(value)) return;
      this.selected.push(value);
    },
    remove(value) {
      this.selected = this.selected.filter((selected) => selected !== value);
    },
    notifyChange() {
      this.$root.closest("form")?.dispatchEvent(new Event("change", { bubbles: true }));
    },
    isSelected(value) {
      return this.selected.includes(value);
    },
    labelFor(value) {
      return this.options.find((option) => option.value === value)?.label || value;
    },
    close() {
      this.open = false;
    },
    triggerKeydown(event) {
      if (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        this.open = true;
        this.$nextTick(() => this.$root.querySelector("[data-multi-select-option]")?.focus());
      }
      if (event.key === "Escape") this.close();
    },
    optionKeydown(event) {
      const options = Array.from(this.$root.querySelectorAll("[data-multi-select-option]:not(:disabled)"));
      const currentIndex = options.indexOf(event.currentTarget);
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        const nextIndex = (currentIndex + (event.key === "ArrowDown" ? 1 : options.length - 1)) % options.length;
        options[nextIndex]?.focus();
      }
      if (event.key === "Escape") {
        event.preventDefault();
        this.close();
        this.$root.querySelector(".multi-select-trigger")?.focus();
      }
    },
  };
};

document.addEventListener("htmx:responseError", (event) => {
  window.dispatchRequestError(event.detail.xhr);
});

document.addEventListener("htmx:sendError", () => {
  window.dispatchEvent(new CustomEvent("snackbar", {
    detail: { type: "error", title: "Connection failed", message: "The server could not be reached." },
  }));
});

if (window.htmx) {
  window.htmx.config.responseHandling = [
    { code: "204", swap: false },
    { code: "[23]..", swap: true },
    { code: "422", swap: true, error: true },
    { code: "[45]xx", swap: false, error: true },
  ];
}
