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

window.catalogSelection = function () {
  return {
    query: "",
    selectedCount: 0,
    initialize() {
      this.refreshCount();
    },
    matches(sku, name) {
      const query = this.query.trim().toLowerCase();
      return !query || `${sku} ${name}`.toLowerCase().includes(query);
    },
    visibleCheckboxes() {
      return Array.from(this.$root.querySelectorAll("[data-product-checkbox]")).filter((checkbox) => checkbox.closest("tr")?.offsetParent !== null);
    },
    allVisibleSelected() {
      const checkboxes = this.visibleCheckboxes();
      return checkboxes.length > 0 && checkboxes.every((checkbox) => checkbox.checked);
    },
    toggleVisible() {
      const select = !this.allVisibleSelected();
      this.visibleCheckboxes().forEach((checkbox) => { checkbox.checked = select; });
      this.refreshCount();
    },
    refreshCount() {
      this.selectedCount = this.$root.querySelectorAll("[data-product-checkbox]:checked").length;
    },
  };
};

window.textDateInput = function () {
  return {
    initialize() {
      const display = this.$root.querySelector("[data-date-display]");
      const canonical = this.$root.querySelector("[data-date-canonical]");
      const picker = this.$root.querySelector("[data-date-picker]");
      const trigger = this.$root.querySelector("[data-date-picker-trigger]");
      if (!display || !canonical) return;

      if (canonical.value && /^\d{4}-\d{2}-\d{2}$/.test(canonical.value)) {
        display.value = this.isoToDisplay(canonical.value);
      } else {
        display.value = this.format(display.value).value;
      }
      this.sync();
      display.addEventListener("input", () => this.formatInput(display));
      display.addEventListener("blur", () => { this.formatInput(display); this.sync(); });
      if (picker) picker.addEventListener("change", () => {
        display.value = this.isoToDisplay(picker.value);
        this.sync();
      });
      if (trigger && picker) trigger.addEventListener("click", () => {
        if (typeof picker.showPicker === "function") picker.showPicker();
        else picker.click();
      });
    },
    format(value, caret) {
      const beforeCaret = typeof caret === "number" ? value.slice(0, caret).replace(/\D/g, "").length : null;
      const digits = value.replace(/\D/g, "").slice(0, 8);
      let formatted = digits;
      if (digits.length > 2) formatted = `${digits.slice(0, 2)}/${digits.slice(2)}`;
      if (digits.length > 4) formatted = `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
      if (beforeCaret === null) return { value: formatted };
      let position = 0;
      let seen = 0;
      while (position < formatted.length && seen < beforeCaret) {
        if (/\d/.test(formatted[position])) seen++;
        position++;
      }
      return { value: formatted, caret: position };
    },
    formatInput(display) {
      const formatted = this.format(display.value, display.selectionStart);
      display.value = formatted.value;
      if (typeof formatted.caret === "number") display.setSelectionRange(formatted.caret, formatted.caret);
      this.sync();
    },
    sync() {
      const display = this.$root.querySelector("[data-date-display]");
      const canonical = this.$root.querySelector("[data-date-canonical]");
      const picker = this.$root.querySelector("[data-date-picker]");
      if (!display || !canonical) return;
      const iso = this.displayToISO(display.value);
      canonical.value = iso;
      if (picker) picker.value = iso;
      canonical.dispatchEvent(new Event("change", { bubbles: true }));
    },
    displayToISO(value) {
      const match = value.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
      if (!match) return "";
      const month = Number(match[1]);
      const day = Number(match[2]);
      const year = Number(match[3]);
      const date = new Date(Date.UTC(year, month - 1, day));
      if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) return "";
      return `${match[3]}-${match[1]}-${match[2]}`;
    },
    isoToDisplay(value) {
      const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
      return match ? `${match[2]}/${match[3]}/${match[1]}` : "";
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
      this.ensureBlankRow();
      this.refreshRows();
      this.refreshSummary();
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
       const tax = row.querySelector("[data-field='tax']");
       if (tax) tax.textContent = this.taxLabel();
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

window.salesOrderForm = function () {
  return {
    initialize() {
      this.refresh();
    },
    addLine() {
      const template = this.$root.querySelector("#order-line-template");
      const container = this.$root.querySelector("#order-lines");
      if (!template || !container) return;
      const index = container.querySelectorAll("[data-order-line]").length;
      container.insertAdjacentHTML("beforeend", template.innerHTML.replaceAll("__INDEX__", index));
      this.refresh();
      container.lastElementChild?.querySelector("select")?.focus();
    },
    moveAll() {
      if (this.$root.querySelector("#order-line-template")) {
        this.addLine();
        return;
      }
      this.$root.querySelectorAll("[data-order-line]").forEach((row) => {
        const input = row.querySelector("[data-quantity]");
        if (input) input.value = row.dataset.remaining || "0";
      });
      this.refresh();
    },
    clearAll() {
      this.$root.querySelectorAll("[data-order-line] input[name$='.quantity']").forEach((input) => input.value = "0");
      this.refresh();
    },
    removeLine(row) {
      const rows = Array.from(this.$root.querySelectorAll("[data-order-line]"));
      const index = rows.indexOf(row);
      if (index >= 0) this.$root.querySelectorAll("#order-line-prices [data-unit-price]")[index]?.remove();
      row?.remove();
      this.refresh();
    },
    refresh() {
      let total = 0;
      let taxRate = 0;
      const prices = Array.from(this.$root.querySelectorAll("#order-line-prices [data-unit-price]"));
      let count = 0;
      const rows = Array.from(this.$root.querySelectorAll("[data-order-line]"));
      rows.forEach((row, index) => {
        const quantity = Number(row.querySelector("input[name$='.quantity']")?.value || 0);
        const priceInfo = prices[index];
        const price = Number(row.querySelector("input[name$='.unit_price']")?.value || row.dataset.rate || priceInfo?.dataset.unitPrice || 0);
        const lineTotal = quantity * price;
        const storedTaxRate = Number(row.querySelector("input[name$='.tax_rate']")?.value || priceInfo?.dataset.taxRate || 0);
        const lineTaxRate = storedTaxRate > 100 ? storedTaxRate / 10000 : storedTaxRate;
        total += lineTotal;
        if (lineTaxRate > 0) taxRate = lineTaxRate;
        if (quantity > 0) count++;
        const amount = row.querySelector("[data-amount]");
        if (amount) amount.textContent = this.money(quantity * price);
        const product = row.querySelector("select[name$='.product_id']");
        const option = product?.selectedOptions[0];
        const uom = row.querySelector("[data-uom]");
        const hiddenUOM = row.querySelector("[data-uom-input]");
        if (option && uom) uom.textContent = option.dataset.uom || "--";
        if (option && hiddenUOM) hiddenUOM.value = option.dataset.uom || "";
      });
      const output = this.$root.querySelector("[data-total]");
      const countOutput = this.$root.querySelector("[data-count]");
      const submit = this.$root.querySelector("[data-submit]");
      const acknowledgement = this.$root.querySelector("[name='edit_acknowledged']");
      const vat = taxRate > 0 ? total - total / (1 + taxRate / 100) : 0;
      if (output) output.textContent = this.money(total);
      const vatOutput = this.$root.querySelector("[data-vat]");
      const netOutput = this.$root.querySelector("[data-net]");
      if (vatOutput) vatOutput.textContent = this.money(vat);
      if (netOutput) netOutput.textContent = this.money(total - vat);
      if (countOutput) countOutput.textContent = `${count} line${count === 1 ? "" : "s"} selected`;
      if (submit) submit.disabled = count === 0 || (acknowledgement && !acknowledgement.checked);
    },
    money(value) { return new Intl.NumberFormat("en-PH", { style: "currency", currency: "PHP" }).format(Number(value || 0)); },
  };
};

window.purchaseOrderForm = function () {
  return {
    openRow: null,
    removedRows: [],
    lineCount: 0,
    initialize() {
      if (this.$root.querySelector("#purchase-sales-product-template") && this.$root.dataset.purchaseEdit !== "true") this.ensureBlankRow();
      if (this.$root.querySelector("#purchase-direct-product-template")) this.ensureBlankRow();
      this.refresh();
      this.repositionOnViewportChange = () => {
        if (this.openRow) this.positionActions(this.openRow);
      };
      window.addEventListener("scroll", this.repositionOnViewportChange, true);
      window.addEventListener("resize", this.repositionOnViewportChange);
    },
    toggleActions(row) {
      this.openRow = this.openRow === row ? null : row;
      if (this.openRow) this.$nextTick(() => this.positionActions(this.openRow));
    },
    closeActions() {
      this.openRow = null;
    },
    removeLine(row) {
      if (!row) return;
      this.removedRows.push(row);
      this.setExcluded(row.dataset.sourceLineId, true);
      row.remove();
      this.closeActions();
      if (this.$root.querySelector("#purchase-direct-product-template")) this.reindexDirectRows();
      this.refresh();
    },
    ensureBlankRow() {
      const container = this.$root.querySelector("#purchase-order-lines");
      const template = this.$root.querySelector("#purchase-sales-product-template");
      const directTemplate = this.$root.querySelector("#purchase-direct-product-template");
      if (directTemplate && !template) {
        if (!container) return;
        const rows = container.querySelectorAll("[data-purchase-line]");
        const last = rows[rows.length - 1];
        if (!last || last.querySelector("input[type='hidden'][name$='.product_id']")?.value) this.appendBlankRow("direct");
        return;
      }
      if (!container || !template) return;
      const rows = container.querySelectorAll("[data-purchase-line]");
      const last = rows[rows.length - 1];
      if (!last || last.dataset.sourceLineId) this.appendBlankRow();
    },
    appendBlankRow(mode = "sales") {
      const template = mode === "direct" ? this.$root.querySelector("#purchase-direct-product-template") : this.$root.querySelector("#purchase-sales-product-template");
      const container = this.$root.querySelector("#purchase-order-lines");
      if (!template || !container) return;
      const index = container.querySelectorAll("[data-purchase-line]").length;
      const wrapper = document.createElement("tbody");
      wrapper.innerHTML = template.innerHTML.replaceAll("__INDEX__", String(index));
      if (wrapper.firstElementChild) container.appendChild(wrapper.firstElementChild);
      this.lineCount = index + 1;
    },
    reindexDirectRows() {
      this.$root.querySelectorAll("#purchase-order-lines [data-purchase-line]").forEach((row, index) => {
        row.querySelectorAll("[name]").forEach((input) => {
          input.name = input.name.replace(/products\[.*?\]/, `products[${index}]`);
        });
        const selector = row.querySelector(".searchable-select");
        if (selector) {
          selector.id = `purchase-product-${index}`;
          selector.querySelector("label")?.setAttribute("for", `purchase-product-${index}-input`);
          selector.querySelector("input[type='hidden']")?.setAttribute("id", `purchase-product-${index}-value`);
          selector.querySelector(".searchable-select-input")?.setAttribute("id", `purchase-product-${index}-input`);
        }
      });
    },
    productSelected(event) {
      const row = event.target.closest("[data-purchase-line]");
      if (!row) return;
      const option = event.detail;
      if (this.$root.querySelector("#purchase-direct-product-template")) {
        row.dataset.productId = option.value;
        row.querySelector("[data-field='uom']").textContent = option.uom || "";
        row.querySelector("[data-field='supplier-sku']").textContent = option.supplierSKU || "";
        const cost = row.querySelector("input[name$='.unit_cost']");
        if (cost) {
          cost.required = true;
          if (option.referenceCost !== undefined) cost.value = option.referenceCost;
        }
        const quantity = row.querySelector("input[name$='.quantity']");
        if (quantity) {
          quantity.min = "0.0001";
          quantity.required = true;
        }
        this.ensureBlankRow();
        this.refresh();
        row.querySelector("input[name$='.quantity']")?.focus();
        return;
      }
      row.dataset.sourceLineId = option.sourceLineID || option.value;
      row.querySelectorAll("[name]").forEach((input) => {
        input.name = input.name.replace(/lines\[[^\]]+\]/, `lines[${option.sourceLineID}]`);
      });
      row.querySelector("[data-field='uom']").textContent = option.uom || "";
      row.querySelector("[data-field='supplier-sku']").textContent = option.supplierSKU || "";
      const product = row.querySelector("input[name$='.product_id']");
      if (product) product.value = option.value;
      const quantity = row.querySelector("input[name$='.quantity']");
      if (quantity && option.quantity) quantity.value = option.quantity;
      const cost = row.querySelector("input[name$='.unit_cost']");
      if (cost && option.referenceCost !== undefined) cost.value = option.referenceCost;
      this.ensureBlankRow();
      this.refresh();
      row.querySelector("input[name$='.quantity']")?.focus();
    },
    restoreLine() {
      const row = this.removedRows.pop();
      const container = this.$root.querySelector("#purchase-order-lines");
      if (!row || !container) return;
      this.setExcluded(row.dataset.sourceLineId, false);
      container.appendChild(row);
      this.refresh();
    },
    addProduct() {
      const template = this.$root.querySelector("#purchase-product-template");
      const container = this.$root.querySelector("#purchase-order-lines");
      if (!template || !container) return;
      const indexes = Array.from(this.$root.querySelectorAll("[name^='products[']"))
        .map((input) => Number(input.name.match(/^products\[(\d+)\]/)?.[1] || 0));
      const index = Math.max(0, ...indexes) + 1;
      const wrapper = document.createElement("tbody");
      wrapper.innerHTML = template.innerHTML.replaceAll("__PRODUCT_ID__", String(index));
      const row = wrapper.firstElementChild;
      if (row) container.appendChild(row);
      this.refresh();
    },
    setExcluded(id, excluded) {
      const existing = this.$root.querySelector(`[data-excluded-line='${id}']`);
      if (excluded && !existing) {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = "excluded_line_ids";
        input.value = id;
        input.dataset.excludedLine = id;
        this.$root.appendChild(input);
      } else if (!excluded) {
        existing?.remove();
      }
    },
    positionActions(row) {
      const button = row?.querySelector(".ellipsis-button");
      const menu = row?.querySelector(".row-action-menu");
      if (!button || !menu) return;
      const buttonBounds = button.getBoundingClientRect();
      const menuBounds = menu.getBoundingClientRect();
      const gap = 4;
      const padding = 8;
      const openAbove = buttonBounds.bottom + menuBounds.height + gap > window.innerHeight - padding;
      const top = openAbove ? buttonBounds.top - menuBounds.height - gap : buttonBounds.bottom + gap;
      const left = Math.min(buttonBounds.right - menuBounds.width, window.innerWidth - menuBounds.width - padding);
      menu.style.top = `${Math.max(padding, top)}px`;
      menu.style.left = `${Math.max(padding, left)}px`;
    },
    refresh() {
      const rows = this.$root.querySelectorAll("#purchase-order-lines [data-purchase-line]");
      const salesOrder = Boolean(this.$root.querySelector("#purchase-sales-product-template"));
      let count = 0;
      let total = 0;
      rows.forEach((row) => {
        const selected = salesOrder
          ? Boolean(row.querySelector("input[name$='.product_id']")?.value)
          : Boolean(row.querySelector("input[name$='.product_id']")?.value || row.dataset.productId);
        if (selected) count++;
        const quantity = Number(row.querySelector("input[name$='.quantity']")?.value || 0);
        const cost = Number(row.querySelector("input[name$='.unit_cost']")?.value || 0);
        const amount = quantity * cost;
        total += amount;
        const output = row.querySelector("[data-amount]");
        if (output) output.textContent = this.money(amount);
      });
      const countOutput = this.$root.querySelector("[data-line-count]");
      const totalOutput = this.$root.querySelector("[data-total]");
      const submit = this.$root.querySelector("[data-submit]");
      if (countOutput) countOutput.textContent = count;
      if (totalOutput) totalOutput.textContent = this.money(total);
      if (submit) submit.disabled = count === 0;
      this.lineCount = count;
    },
    money(value) { return new Intl.NumberFormat("en-PH", { style: "currency", currency: "PHP" }).format(Number(value || 0)); },
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
        supplierSKU: option.dataset.supplierSku || "",
        referenceCost: option.dataset.referenceCost || "",
        sourceLineID: option.dataset.sourceLineId || "",
        quantity: option.dataset.quantity || "",
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
      this.$root.closest("[data-purchase-line]")?.removeAttribute("data-product-id");
      this.$root.querySelector("input[type='hidden']").value = "";
      this.$root.querySelector(".searchable-select-input").value = "";
      const row = this.$root.closest("[data-purchase-line]");
      if (row) {
        const quantity = row.querySelector("input[name$='.quantity']");
        const cost = row.querySelector("input[name$='.unit_cost']");
        if (quantity) {
          quantity.min = "0";
          quantity.required = false;
        }
        if (cost) cost.required = false;
      }
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

window.receivableInvoiceForm = function () {
  return {
    invoiceID: "",
    invoiceNumber: "",
    initialize() {
      const select = this.$root.querySelector("#invoice_id");
      const input = this.$root.querySelector("#invoice_number");
      if (!select || !input) return;
      this.invoiceID = select.value;
      this.invoiceNumber = input.value.trim();
      if (this.invoiceID) {
        this.copyInvoiceNumber(select, input);
        return;
      }
      const matching = Array.from(select.options).find((option) => option.dataset.invoiceNumber === this.invoiceNumber);
      if (matching) {
        select.value = matching.value;
        this.invoiceID = matching.value;
      }
    },
    invoiceChanged(event) {
      if (event.target?.id !== "invoice_id") return;
      const input = this.$root.querySelector("#invoice_number");
      if (!input) return;
      this.invoiceID = event.target.value;
      if (this.invoiceID) {
        this.copyInvoiceNumber(event.target, input);
      } else {
        this.invoiceNumber = "";
        input.value = "";
      }
    },
    invoiceNumberChanged(event) {
      if (event.target?.id !== "invoice_number" || this.invoiceID) return;
      const select = this.$root.querySelector("#invoice_id");
      if (!select) return;
      this.invoiceNumber = event.target.value.trim();
      const matching = Array.from(select.options).find((option) => option.dataset.invoiceNumber === this.invoiceNumber);
      select.value = matching ? matching.value : "";
      this.invoiceID = select.value;
    },
    copyInvoiceNumber(select, input) {
      const option = select.options[select.selectedIndex];
      this.invoiceNumber = option?.dataset.invoiceNumber || "";
      input.value = this.invoiceNumber;
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
