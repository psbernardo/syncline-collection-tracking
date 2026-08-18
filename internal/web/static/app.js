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

window.quotationForm = function () {
  return {
    lineCount: 0,
    quotationTax: "NONE",
    initialize() {
      this.lineCount = document.querySelectorAll("#quotation-lines [data-line]").length;
      this.quotationTax = this.$root.querySelector("[name='tax_default_code']")?.value || "NONE";
      this.initializeDatePicker();
      this.ensureBlankRow();
      this.refreshRows();
      this.refreshSummary();
    },
    initializeDatePicker() {
      const display = this.$root.querySelector("[data-date-display]");
      const picker = this.$root.querySelector("[data-date-picker]");
      const trigger = this.$root.querySelector("[data-date-picker-trigger]");
      if (!display || !picker || !trigger) return;
      display.addEventListener("input", () => this.syncDatePicker());
      display.addEventListener("blur", () => this.syncDatePicker());
      picker.addEventListener("change", () => {
        display.value = this.formatDatePickerValue(picker.value);
      });
      trigger.addEventListener("click", () => {
        if (typeof picker.showPicker === "function") picker.showPicker();
        else picker.click();
      });
    },
    syncDatePicker() {
      const display = this.$root.querySelector("[data-date-display]");
      const picker = this.$root.querySelector("[data-date-picker]");
      if (!display || !picker) return;
      const match = display.value.trim().match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
      if (!match) {
        picker.value = "";
        return;
      }
      const month = Number(match[1]);
      const day = Number(match[2]);
      const year = Number(match[3]);
      const date = new Date(Date.UTC(year, month - 1, day));
      if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
        picker.value = "";
        return;
      }
      picker.value = `${match[3]}-${match[1]}-${match[2]}`;
    },
    formatDatePickerValue(value) {
      const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
      return match ? `${match[2]}/${match[3]}/${match[1]}` : "";
    },
    ensureBlankRow() {
      const rows = document.querySelectorAll("#quotation-lines [data-line]");
      if (rows.length === 0 || rows[rows.length - 1].querySelector("input[type='hidden'][name$='.product_id']")?.value) this.appendBlankRow();
    },
    appendBlankRow() {
      const container = document.querySelector("#quotation-lines");
      const index = container.querySelectorAll("[data-line]").length;
      container.insertAdjacentHTML("beforeend", document.querySelector("#quotation-row-template").innerHTML.replaceAll("__INDEX__", index));
      this.lineCount = index + 1;
    },
    productSelected(event) {
      const selector = event.target;
      const row = selector.closest("[data-line]");
      if (!row) return;
      const option = event.detail;
      row.querySelector("[data-field='uom']").textContent = option.uom;
      row.querySelector("[data-field='uom-input']").value = option.uom;
      this.ensureBlankRow();
      this.refreshRows();
      row.querySelector("[data-row-input='quantity']")?.focus();
    },
    fieldChanged(event) {
      const row = event.target.closest("[data-line]");
      if (!row) { this.refreshRows(); return; }
      const field = event.target.dataset.rowInput;
      if (!field) { this.refreshRow(row); return; }
       const hiddenField = field === "quantity" ? "quantity-input" : field === "rate" ? "rate-input" : "supplier-cost-input";
       const hidden = row.querySelector(`[data-field='${hiddenField}']`);
      if (hidden) hidden.value = event.target.value;
      this.refreshRow(row);
      this.refreshSummary();
      if (event.key === "Tab" && field === "rate" && row === document.querySelectorAll("#quotation-lines [data-line]")[document.querySelectorAll("#quotation-lines [data-line]").length - 2]) this.ensureBlankRow();
    },
    applyTaxToRows() { this.refreshRows(); },
    taxRate() { return this.quotationTax === "NONE" ? 0 : 12; },
    refreshRows() { document.querySelectorAll("#quotation-lines [data-line]").forEach((row, index) => { row.querySelector("[data-field='row-number']").textContent = row.querySelector("input[type='hidden'][name$='.product_id']")?.value ? index + 1 : ""; this.refreshRow(row); }); this.refreshSummary(); },
    refreshRow(row) {
      const quantity = Number(row.querySelector("[data-field='quantity']")?.value || 0);
      const rate = Number(row.querySelector("[data-field='rate']")?.value || 0);
       const subtotal = quantity * rate;
       const cost = quantity * Number(row.querySelector("[data-field='supplier-cost']")?.value || 0);
       const commission = this.lineCommission(row, subtotal);
       const profit = subtotal - cost - commission;
      row.querySelector("[data-field='tax']").textContent = this.taxLabel();
       row.querySelector("[data-field='amount']").textContent = this.money(subtotal);
       row.querySelector("[data-field='profit']").innerHTML = `${this.money(profit)}<br><small data-field="margin">${this.percent(subtotal ? profit / subtotal * 100 : 0)}</small>`;
      row.querySelector("input[name$='.tax_code']").value = this.quotationTax;
      row.querySelector("input[name$='.tax_rate']").value = this.taxRate();
    },
    netSubtotal() { let subtotal = 0; document.querySelectorAll("#quotation-lines [data-line]").forEach((row) => { subtotal += Number(row.querySelector("[data-field='quantity']")?.value || 0) * Number(row.querySelector("[data-field='rate']")?.value || 0); }); return subtotal; },
    vatAmount() { const gross = this.netSubtotal(); return this.taxRate() === 0 ? 0 : gross - gross / 1.12; },
     // Quotation rates are VAT-inclusive, so VAT is extracted from the total rather than added to it.
     totalAmount() { return this.netSubtotal(); },
    commissionAmount() { const rate = Number(this.$root.querySelector("[name='commission_rate']")?.value || 0); const type = this.$root.querySelector("[name='commission_type']")?.value || "NONE"; if (type === "PER_UNIT") { let value = 0; document.querySelectorAll("#quotation-lines [data-line]").forEach((row) => { value += Number(row.querySelector("[data-field='quantity']")?.value || 0) * rate; }); return value; } if (type === "PERCENTAGE") return this.netOfVAT() * rate / 100; if (type === "FIXED_QUOTATION") return rate; return 0; },
    lineCommission(row, subtotal) { const total = this.netSubtotal(); const rate = Number(this.$root.querySelector("[name='commission_rate']")?.value || 0); const type = this.$root.querySelector("[name='commission_type']")?.value || "NONE"; if (type === "PER_UNIT") return Number(row.querySelector("[data-field='quantity']")?.value || 0) * rate; if (type === "PERCENTAGE") return subtotal / (total || 1) * this.commissionAmount(); if (type === "FIXED_QUOTATION") return subtotal / (total || 1) * this.commissionAmount(); return 0; },
    percent(value) { return `${Number(value || 0).toFixed(2)}%`; },
    netOfVAT() { return this.netSubtotal() - this.vatAmount(); },
    summaryTotal() { return this.money(this.totalAmount()); },
    refreshSummary() { const net = this.netOfVAT(); const costs = this.quoteCost("supplier_delivery_cost") + this.quoteCost("customer_delivery_cost") + this.quoteCost("other_cost"); const supplier = Array.from(document.querySelectorAll("#quotation-lines [data-line]")).reduce((total, row) => total + Number(row.querySelector("[data-field='quantity']")?.value || 0) * Number(row.querySelector("[data-field='supplier-cost']")?.value || 0), 0); const profit = net - supplier - costs - this.commissionAmount(); const values = { total: this.totalAmount(), vat: this.vatAmount(), net, cost: supplier + costs + this.commissionAmount(), profit, margin: this.percent(net ? profit / net * 100 : 0) }; Object.entries(values).forEach(([key, value]) => { const element = this.$root.querySelector(`[data-summary='${key}']`); if (element) element.textContent = key === "margin" ? value : this.money(value); }); },
    quoteCost(name) { return Number(this.$root.querySelector(`[name='${name}']`)?.value || 0); },
    taxLabel() { return { NONE: "No tax", VAT12: "VAT-inclusive, 12% VAT", VAT_EWT1: "VAT-inclusive, 1% EWT" }[this.quotationTax] || "No tax"; },
    money(value) { return new Intl.NumberFormat("en-PH", { style: "currency", currency: "PHP" }).format(Number(value || 0)); },
    removeLine(line) {
      const container = document.querySelector("#quotation-lines");
      line.remove();
      this.ensureBlankRow();
      this.lineCount = container.querySelectorAll("[data-line]").length;
      container.querySelectorAll("[data-line]").forEach((item, index) => {
        item.querySelectorAll("[name]").forEach((input) => {
          input.name = input.name.replace(/lines\[\d+\]/, `lines[${index}]`);
        });
      });
      this.refreshRows();
    },
  };
};

window.searchableSelect = function () {
  return {
    open: false,
    query: "",
    selected: "",
    options: [],
    initialize() {
      this.options = Array.from(this.$root.querySelectorAll("[data-value]")).map((option) => ({
        value: option.dataset.value,
        label: option.dataset.label,
        search: option.dataset.search || option.dataset.label,
        uom: option.dataset.uom || "",
        disabled: option.dataset.disabled === "true",
      }));
      this.selected = this.$root.dataset.selected || "";
      const selected = this.options.find((option) => option.value === this.selected);
      if (selected) {
        this.$root.querySelector(".searchable-select-input").value = selected.label;
        this.setUOM(selected.uom);
      }
      this.repositionOnScroll = () => { if (this.open) this.positionOptions(); };
      window.addEventListener("scroll", this.repositionOnScroll, true);
      window.addEventListener("resize", this.repositionOnScroll);
    },
    filteredOptions() {
      const query = this.query.trim().toLowerCase();
      return this.options.filter((option) => !query || option.search.toLowerCase().includes(query));
    },
    isVisible(value) {
      return this.filteredOptions().some((option) => option.value === value);
    },
    positionOptions() {
      const control = this.$root.querySelector(".searchable-select-control");
      const options = this.$root.querySelector(".searchable-select-options");
      if (!control || !options) return;
      const bounds = control.getBoundingClientRect();
      const gap = 4;
      const viewportPadding = 12;
      const availableBelow = window.innerHeight - bounds.bottom - viewportPadding;
      const availableAbove = bounds.top - viewportPadding;
      const openAbove = availableBelow < 180 && availableAbove > availableBelow;
      options.classList.add("is-floating");
      options.style.left = `${bounds.left}px`;
      options.style.width = `${bounds.width}px`;
      options.style.top = openAbove ? "auto" : `${bounds.bottom + gap}px`;
      options.style.bottom = openAbove ? `${window.innerHeight - bounds.top + gap}px` : "auto";
      options.style.maxHeight = `${Math.max(120, openAbove ? availableAbove : availableBelow)}px`;
    },
    select(value) {
      const option = this.options.find((item) => item.value === value);
      if (!option || option.disabled) return;
      this.selected = value;
      this.query = option.label;
      this.$root.querySelector("input[type='hidden']").value = value;
      this.$root.querySelector(".searchable-select-input").value = option.label;
      this.setUOM(option.uom);
      this.$root.dispatchEvent(new CustomEvent("product-selected", { detail: option, bubbles: true }));
      this.close();
      this.$root.closest("form")?.dispatchEvent(new Event("change", { bubbles: true }));
    },
    clear() {
      this.selected = "";
      this.query = "";
      this.$root.querySelector("input[type='hidden']").value = "";
      this.$root.querySelector(".searchable-select-input").value = "";
      this.setUOM("");
      this.open = true;
    },
    setUOM(value) {
      const input = this.$root.closest(".quotation-modal")?.querySelector("[data-uom-input]") || this.$root.closest("[data-line], form")?.querySelector("[data-uom-input]");
      if (input) { input.value = value; input.dispatchEvent(new Event("input", { bubbles: true })); }
    },
    close() { this.open = false; },
    inputKeydown(event) {
      if (event.key === "Escape") { event.preventDefault(); this.close(); return; }
      if (event.key === "Enter") {
        event.preventDefault();
        const option = this.filteredOptions()[0];
        if (option) this.select(option.value);
      }
      if (event.key === "ArrowDown") {
        event.preventDefault(); this.open = true;
        this.$nextTick(() => this.$root.querySelector(".searchable-select-option:not(:disabled)")?.focus());
      }
    },
    optionKeydown(event) {
      const options = Array.from(this.$root.querySelectorAll(".searchable-select-option:not(:disabled)"))
        .filter((option) => option.offsetParent !== null);
      const current = options.indexOf(event.currentTarget);
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        const next = (current + (event.key === "ArrowDown" ? 1 : options.length - 1)) % options.length;
        options[next]?.focus();
      }
      if (event.key === "Escape") {
        event.preventDefault();
        this.close();
        this.$root.querySelector(".searchable-select-input")?.focus();
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
