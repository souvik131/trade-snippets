class MarketDataUI {
  constructor() {
    this.app = document.getElementById("app");
    this.status = document.getElementById("status");
    this.marketStatus = document.querySelector(".market-status");
    this.expirySelect = document.getElementById("expirySelect");
    this.stockCount = document.getElementById("stockCount");
    this.stockData = new Map();
    this.decoder = new TextDecoder();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
    this.currentSort = { column: "name", direction: "asc" };
    this.selectedExpiry = null;
    this.debug = false;
    this.setupSortingListeners();
    this.setupExpirySelect();
    this.connect();
    this.startStaleDataCheck();
  }

  calculateImpliedVolatility(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const ceMidPrice = (stock.ce.bid + stock.ce.ask) / 2;
    const peMidPrice = (stock.pe.bid + stock.pe.ask) / 2;

    const now = new Date();
    const expiry = new Date(stock.expiry);
    expiry.setHours(15, 30, 0, 0);

    const totalTimeInSeconds = this.calculateTradingSeconds(now, expiry);
    const leftTimeInSeconds = this.calculateTradingSeconds(now, expiry);

    if (totalTimeInSeconds <= 0 || leftTimeInSeconds <= 0) return null;

    const iv =
      1.25 *
      ((ceMidPrice + peMidPrice) / stock.future) *
      Math.sqrt(totalTimeInSeconds / leftTimeInSeconds) *
      100;

    return iv;
  }

  calculateBidIv(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const now = new Date();
    const expiry = new Date(stock.expiry);
    expiry.setHours(15, 30, 0, 0);

    const totalTimeInSeconds = this.calculateTradingSeconds(now, expiry);
    const leftTimeInSeconds = this.calculateTradingSeconds(now, expiry);

    if (totalTimeInSeconds <= 0 || leftTimeInSeconds <= 0) return null;

    const iv =
      1.25 *
      ((stock.ce.bid + stock.pe.bid) / stock.future) *
      Math.sqrt(totalTimeInSeconds / leftTimeInSeconds) *
      100;

    return iv;
  }

  calculateAskIv(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const now = new Date();
    const expiry = new Date(stock.expiry);
    expiry.setHours(15, 30, 0, 0);

    const totalTimeInSeconds = this.calculateTradingSeconds(now, expiry);
    const leftTimeInSeconds = this.calculateTradingSeconds(now, expiry);

    if (totalTimeInSeconds <= 0 || leftTimeInSeconds <= 0) return null;

    const iv =
      1.25 *
      ((stock.ce.ask + stock.pe.ask) / stock.future) *
      Math.sqrt(totalTimeInSeconds / leftTimeInSeconds) *
      100;

    return iv;
  }

  calculateSpreadIV(stock) {
    if (!stock || !stock.ce || !stock.pe) return null;
    return (
      (stock.ce.askIv - stock.ce.bidIv + (stock.pe.askIv - stock.pe.bidIv)) / 2
    );
  }

  calculateSpreadIV(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const bidIv = this.calculateBidIv(stock);
    const askIv = this.calculateAskIv(stock);

    if (bidIv === null || askIv === null) return null;

    return askIv - bidIv;
  }

  calculateRV(stock) {
    this.setupExpirySelect();
    this.connect();
    this.startStaleDataCheck();
  }

  calculateBidIv(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const now = new Date();
    const expiry = new Date(stock.expiry);
    expiry.setHours(15, 30, 0, 0);

    const totalTimeInSeconds = this.calculateAnnualTradingSeconds(); // Annual trading seconds
    const leftTimeInSeconds = this.calculateRemainingTradingSeconds(
      now,
      expiry
    ); // Time remaining until expiry
    if (totalTimeInSeconds <= 0 || leftTimeInSeconds <= 0) return null;

    const iv =
      1.25 *
      ((stock.ce.bid + stock.pe.bid) / stock.future) *
      Math.sqrt(totalTimeInSeconds / leftTimeInSeconds) *
      100;

    return iv;
  }

  calculateAskIv(stock) {
    if (!stock || !stock.ce || !stock.pe || !stock.future || !stock.expiry)
      return null;

    const now = new Date();
    const expiry = new Date(stock.expiry);
    expiry.setHours(15, 30, 0, 0);

    const totalTimeInSeconds = this.calculateAnnualTradingSeconds(); // Annual trading seconds
    const leftTimeInSeconds = this.calculateRemainingTradingSeconds(
      now,
      expiry
    ); // Time remaining until expiry
    if (totalTimeInSeconds <= 0 || leftTimeInSeconds <= 0) return null;

    const iv =
      1.25 *
      ((stock.ce.ask + stock.pe.ask) / stock.future) *
      Math.sqrt(totalTimeInSeconds / leftTimeInSeconds) *
      100;

    return iv;
  }

  // Calculate annual trading seconds assuming 252 trading days per year and 6.5 hours per trading day
  calculateAnnualTradingSeconds() {
    const tradingHoursPerDay = 6.5 * 3600; // 6.5 hours in seconds
    const tradingDaysPerYear = 252; // Number of trading days in a year
    return tradingHoursPerDay * tradingDaysPerYear;
  }

  calculateRemainingTradingSeconds(now, expiry) {
    const tradingStartInSeconds = 9 * 3600 + 15 * 60; // 9:15 AM in seconds
    const tradingEndInSeconds = 15 * 3600 + 30 * 60; // 3:30 PM in seconds
    const tradingHoursPerDay = 6.5 * 3600; // 6.5 hours in seconds
    let remainingSeconds = 0;

    let currentDate = new Date(
      now.getFullYear(),
      now.getMonth(),
      now.getDate()
    );

    while (currentDate <= expiry) {
      // If it's a weekday (Monday to Friday)
      const dayOfWeek = currentDate.getDay();
      if (dayOfWeek > 0 && dayOfWeek < 6) {
        // 0=Sunday, 6=Saturday
        if (
          currentDate.toDateString() === now.toDateString() // Same day as "now"
        ) {
          const currentTimeInSeconds =
            now.getHours() * 3600 + now.getMinutes() * 60 + now.getSeconds();

          if (currentTimeInSeconds < tradingStartInSeconds) {
            // If current time is before trading starts, add the full trading day
            remainingSeconds += tradingHoursPerDay;
          } else if (currentTimeInSeconds < tradingEndInSeconds) {
            // If current time is during trading hours, add remaining time
            remainingSeconds += tradingEndInSeconds - currentTimeInSeconds;
          }
        } else if (currentDate.toDateString() === expiry.toDateString()) {
          // On expiry day, add only up to the expiry time (15:30)
          remainingSeconds += tradingEndInSeconds;
        } else {
          // For other weekdays, add a full trading day
          remainingSeconds += tradingHoursPerDay;
        }
      }

      // Move to the next day
      currentDate.setDate(currentDate.getDate() + 1);
    }

    return remainingSeconds;
  }

  calculateRV(stock) {
    return (
      Math.abs(this.calculateChange(stock.future, stock.futureClose)) *
        Math.sqrt(252) || null
    );
  }

  formatIV(value) {
    if (value === null || isNaN(value)) return "".padStart(8);
    return value.toFixed(2).padStart(8);
  }

  setupSortingListeners() {
    const headers = document.querySelectorAll(".header > div");
    headers.forEach((header) => {
      header.addEventListener("click", () => {
        const column = header.dataset.sort;
        if (this.currentSort.column === column) {
          this.currentSort.direction =
            this.currentSort.direction === "asc" ? "desc" : "asc";
        } else {
          this.currentSort.column = column;
          this.currentSort.direction = "asc";
        }

        headers.forEach((h) => {
          h.classList.remove("sort-asc", "sort-desc");
          if (h.dataset.sort === this.currentSort.column) {
            h.classList.add(
              this.currentSort.direction === "asc" ? "sort-asc" : "sort-desc"
            );
          }
        });

        this.sortAndRenderRows();
      });
    });
  }

  setupExpirySelect() {
    this.expirySelect.addEventListener("change", () => {
      this.selectedExpiry = this.expirySelect.value;
      this.sortAndRenderRows();
    });
  }

  updateExpiryOptions() {
    const expiries = new Set();
    this.stockData.forEach((stock) => {
      if (stock.expiry) {
        expiries.add(stock.expiry);
      }
    });

    const sortedExpiries = Array.from(expiries).sort();

    if (!this.selectedExpiry && sortedExpiries.length > 0) {
      this.selectedExpiry = sortedExpiries[0];
    }

    this.expirySelect.innerHTML = "";

    sortedExpiries.forEach((expiry) => {
      const option = document.createElement("option");
      option.value = expiry;
      option.textContent = this.formatExpiry(expiry);
      this.expirySelect.appendChild(option);
    });

    this.expirySelect.value = this.selectedExpiry;
    this.updateStockCount();
  }

  updateStockCount() {
    const currentExpiryStocks = Array.from(this.stockData.entries()).filter(
      ([_, data]) => data.expiry === this.selectedExpiry
    ).length;
    this.stockCount.textContent = `Stocks: ${currentExpiryStocks}`;
  }

  formatExpiry(expiry) {
    if (!expiry) return "".padStart(8);
    const date = new Date(expiry);
    return date
      .toLocaleDateString("en-IN", {
        day: "2-digit",
        month: "short",
        year: "2-digit",
      })
      .replace(",", "")
      .padStart(8);
  }

  formatIndianCurrency(number) {
    if (!number || isNaN(number)) return "".padStart(8);

    const absNumber = Math.abs(number);
    let result;
    if (absNumber >= 10000000) {
      result = (number / 10000000).toFixed(2) + "Cr";
    } else if (absNumber >= 100000) {
      result = (number / 100000).toFixed(2) + "L";
    } else {
      result = number.toFixed(2);
    }
    return result.padStart(8);
  }

  formatVolume(number) {
    if (!number || isNaN(number)) return "".padStart(8);

    const absNumber = Math.abs(number);
    let result;
    if (absNumber >= 10000000) {
      result = (absNumber / 10000000).toFixed(2) + "Cr";
    } else if (absNumber >= 100000) {
      result = (absNumber / 100000).toFixed(2) + "L";
    } else if (absNumber >= 1000) {
      result = absNumber.toLocaleString("en-IN");
    } else {
      result = absNumber.toString();
    }
    return result.padStart(8);
  }

  calculateChange(current, close) {
    if (!current || !close || close === 0) return null;
    return ((current - close) / close) * 100;
  }

  formatChange(change) {
    if (change === null) return "".padStart(8);
    const formatted = change.toFixed(2).padStart(6);
    const color = change >= 0 ? "positive" : "negative";
    return `<span class="${color}">${formatted}%</span>`;
  }

  formatPrice(price) {
    if (!price || isNaN(price)) return "".padStart(8);
    return price.toFixed(2).padStart(8);
  }

  formatBidAsk(bid, ask) {
    const bidStr = bid ? bid.toFixed(2).padStart(6) : "".padStart(6);
    const askStr = ask ? ask.toFixed(2).padStart(6) : "".padStart(6);
    return `<div class="bid-ask"><span class="bid">${bidStr}</span><span class="ask">${askStr}</span></div>`;
  }

  createStockRow(key, stockName) {
    const row = document.createElement("div");
    row.id = `stock-${key}`;
    row.className = "stock-row";
    row.innerHTML = `
      <div>${stockName.padEnd(12)}</div>
      <div class="future">${"\u00A0".repeat(8)}</div>
      <div class="change">${"\u00A0".repeat(8)}</div>
      <div class="strike">${"\u00A0".repeat(8)}</div>
      <div class="expiry">${"\u00A0".repeat(8)}</div>
      <div class="ce"><div class="bid-ask"><span class="bid">${"\u00A0".repeat(
        6
      )}</span><span class="ask">${"\u00A0".repeat(6)}</span></div></div>
      <div class="ce-oi">${"\u00A0".repeat(8)}</div>
      <div class="ce-vol">${"\u00A0".repeat(8)}</div>
      <div class="pe"><div class="bid-ask"><span class="bid">${"\u00A0".repeat(
        6
      )}</span><span class="ask">${"\u00A0".repeat(6)}</span></div></div>
      <div class="pe-oi">${"\u00A0".repeat(8)}</div>
      <div class="pe-vol">${"\u00A0".repeat(8)}</div>
      <div class="bid-iv">${"\u00A0".repeat(8)}</div>
      <div class="ask-iv">${"\u00A0".repeat(8)}</div>
      <div class="spread-iv">${"\u00A0".repeat(8)}</div>
      <div class="rv">${"\u00A0".repeat(8)}</div>
      <div class="vrp">${"\u00A0".repeat(8)}</div>
      <div class="iv-ratio">${"\u00A0".repeat(8)}</div>
      <div class="lot-size">${"\u00A0".repeat(8)}</div>
      <div class="exposure">${"\u00A0".repeat(8)}</div>
    `;
    this.app.appendChild(row);
  }

  // Add method to find next expiry data
  findNextExpiryData(stockName, currentExpiry) {
    const nextExpiry = Array.from(this.stockData.entries())
      .filter(
        ([_, data]) =>
          data.stockName === stockName && data.expiry > currentExpiry
      )
      .sort((a, b) => a[1].expiry.localeCompare(b[1].expiry))[0];

    return nextExpiry ? nextExpiry[1] : null;
  }

  // Add method to calculate IV ratio
  calculateIVRatio(stock) {
    if (!stock) return null;

    const currentIV =
      (this.calculateBidIv(stock) + this.calculateAskIv(stock)) / 2;
    if (!currentIV) return null;

    const nextExpiryData = this.findNextExpiryData(
      stock.stockName,
      stock.expiry
    );
    if (!nextExpiryData) return null;

    const nextExpiryIV =
      (this.calculateBidIv(nextExpiryData) +
        this.calculateAskIv(nextExpiryData)) /
      2;
    if (!nextExpiryIV) return null;

    return currentIV / nextExpiryIV;
  }

  updateRow(key) {
    const row = document.getElementById(`stock-${key}`);
    const stock = this.stockData.get(key);
    if (!row || !stock) return;

    const futureEl = row.querySelector(".future");
    const changeEl = row.querySelector(".change");
    const strikeEl = row.querySelector(".strike");
    const expiryEl = row.querySelector(".expiry");
    const ceEl = row.querySelector(".ce");
    const ceOiEl = row.querySelector(".ce-oi");
    const ceVolEl = row.querySelector(".ce-vol");
    const peEl = row.querySelector(".pe");
    const peOiEl = row.querySelector(".pe-oi");
    const peVolEl = row.querySelector(".pe-vol");
    const bidIvEl = row.querySelector(".bid-iv");
    const askIvEl = row.querySelector(".ask-iv");
    const spreadIvEl = row.querySelector(".spread-iv");
    const rvEl = row.querySelector(".rv");
    const vrpEl = row.querySelector(".vrp");
    const ivRatioEl = row.querySelector(".iv-ratio");
    const lotEl = row.querySelector(".lot-size");
    const expEl = row.querySelector(".exposure");

    if (stock.future !== null && stock.future > 0) {
      futureEl.textContent = this.formatPrice(stock.future);
      const change = this.calculateChange(stock.future, stock.futureClose);
      changeEl.innerHTML = this.formatChange(change);
    }

    if (stock.strike !== null && stock.strike > 0) {
      strikeEl.textContent = this.formatPrice(stock.strike);
    }

    if (stock.expiry) {
      expiryEl.textContent = this.formatExpiry(stock.expiry);
    }

    if (stock.ce) {
      ceEl.innerHTML = this.formatBidAsk(stock.ce.bid, stock.ce.ask);
      ceOiEl.textContent = this.formatVolume(
        Math.round(stock.ce.oi / stock.lotSize)
      );
      ceVolEl.textContent = this.formatVolume(
        Math.round(stock.ce.volumeTraded / stock.lotSize)
      );
    }

    if (stock.pe) {
      peEl.innerHTML = this.formatBidAsk(stock.pe.bid, stock.pe.ask);
      peOiEl.textContent = this.formatVolume(
        Math.round(stock.pe.oi / stock.lotSize)
      );
      peVolEl.textContent = this.formatVolume(
        Math.round(stock.pe.volumeTraded / stock.lotSize)
      );
    }

    const bidIv = this.calculateBidIv(stock);
    const askIv = this.calculateAskIv(stock);
    const spreadIv = this.calculateSpreadIV(stock);
    const rv = this.calculateRV(stock);
    const vrp = (bidIv + askIv) / rv;
    const ivRatio = this.calculateIVRatio(stock);

    ivRatioEl.textContent = this.formatIV(ivRatio);

    bidIvEl.textContent = this.formatIV(bidIv);
    askIvEl.textContent = this.formatIV(askIv);
    spreadIvEl.textContent = this.formatIV(spreadIv);
    rvEl.textContent = this.formatIV(rv);
    vrpEl.textContent = this.formatIV(vrp);

    if (stock.lotSize > 0) {
      lotEl.textContent = stock.lotSize.toString().padStart(8);
    }

    if (stock.lotSize > 0 && stock.future > 0) {
      const exposure = stock.lotSize * stock.future;
      expEl.textContent = this.formatIndianCurrency(exposure);
    }
  }

  sortAndRenderRows() {
    const rows = Array.from(
      document.querySelectorAll(".stock-row:not(.header)")
    );
    const sortedStocks = Array.from(this.stockData.entries())
      .filter(([_, data]) => {
        return data.expiry === this.selectedExpiry;
      })
      .sort((a, b) => {
        const [_, dataA] = a;
        const [__, dataB] = b;
        const nameA = dataA.stockName;
        const nameB = dataB.stockName;

        let comparison = 0;
        switch (this.currentSort.column) {
          case "name":
            comparison = nameA.localeCompare(nameB);
            break;
          case "future":
            comparison = (dataA.future || 0) - (dataB.future || 0);
            break;
          case "change":
            const changeA =
              this.calculateChange(dataA.future, dataA.futureClose) || 0;
            const changeB =
              this.calculateChange(dataB.future, dataB.futureClose) || 0;
            comparison = changeA - changeB;
            break;
          case "strike":
            comparison = (dataA.strike || 0) - (dataB.strike || 0);
            break;
          case "expiry":
            comparison = (dataA.expiry || "").localeCompare(dataB.expiry || "");
            break;
          case "lot":
            comparison = (dataA.lotSize || 0) - (dataB.lotSize || 0);
            break;
          case "ce":
            comparison = (dataA.ce?.bid || 0) - (dataB.ce?.bid || 0);
            break;
          case "ce_oi":
            comparison =
              Math.round(dataA.ce?.oi / dataA.lotSize || 0) -
              Math.round(dataB.ce?.oi / dataB.lotSize || 0);
            break;
          case "ce_vol":
            comparison =
              Math.round(dataA.ce?.volumeTraded / dataA.lotSize || 0) -
              Math.round(dataB.ce?.volumeTraded / dataB.lotSize || 0);
            break;
          case "pe":
            comparison = (dataA.pe?.bid || 0) - (dataB.pe?.bid || 0);
            break;
          case "pe_oi":
            comparison =
              Math.round(dataA.pe?.oi / dataA.lotSize || 0) -
              Math.round(dataB.pe?.oi / dataB.lotSize || 0);
            break;
          case "pe_vol":
            comparison =
              Math.round(dataA.pe?.volumeTraded || 0) -
              (dataB.pe?.volumeTraded || 0);
            break;
          case "bid_iv":
            comparison =
              (this.calculateBidIv(dataA) || 0) -
              (this.calculateBidIv(dataB) || 0);
            break;
          case "ask_iv":
            comparison =
              (this.calculateAskIv(dataA) || 0) -
              (this.calculateAskIv(dataB) || 0);
            break;
          case "spread_iv":
            comparison =
              (this.calculateSpreadIV(dataA) || 0) -
              (this.calculateSpreadIV(dataB) || 0);
            break;
          case "iv_ratio":
            comparison =
              (this.calculateIVRatio(dataA) || 0) -
              (this.calculateIVRatio(dataB) || 0);
            break;
          case "rv":
            comparison =
              (this.calculateRV(dataA) || 0) - (this.calculateRV(dataB) || 0);
            break;
          case "vrp":
            comparison =
              ((this.calculateAskIv(dataA) + this.calculateBidIv(dataA)) /
                this.calculateRV(dataA) || 0) -
              ((this.calculateAskIv(dataB) + this.calculateBidIv(dataB)) /
                this.calculateRV(dataB) || 0);
            break;
          case "exposure":
            const expA = (dataA.lotSize || 0) * (dataA.future || 0);
            const expB = (dataB.lotSize || 0) * (dataB.future || 0);
            comparison = expA - expB;
            break;
        }

        return this.currentSort.direction === "asc" ? comparison : -comparison;
      });

    rows.forEach((row) => row.remove());

    sortedStocks.forEach(([key, data]) => {
      this.createStockRow(key, data.stockName);
      this.updateRow(key);
    });

    this.updateStockCount();
  }

  startStaleDataCheck() {
    setInterval(() => this.checkStaleData(), 2000);
  }

  checkStaleData() {
    const now = new Date();
    const marketOpen =
      now.getHours() >= 9 &&
      (now.getHours() < 15 || (now.getHours() == 15 && now.getMinutes() < 30));
    this.marketStatus.textContent = marketOpen
      ? "Market Open"
      : "Market Closed";

    this.stockData.forEach((stock, key) => {
      const row = document.getElementById(`stock-${key}`);
      if (!row) return;

      const isStale = now - (stock.lastUpdate || 0) > 2000;
      row.classList.toggle("stale", isStale && marketOpen);
    });
  }

  updateStatus(state) {
    this.status.className = state;
    this.status.textContent = state.charAt(0).toUpperCase() + state.slice(1);
  }

  connect() {
    this.updateStatus("connecting");
    console.log("Connecting to WebSocket server...");
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const domain = window.location.host;
    const wsUrl = `${protocol}://${domain}/ws`;

    this.ws = new WebSocket(wsUrl);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.updateStatus("connected");
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
    };

    this.ws.onmessage = (event) => {
      try {
        this.handleBinaryMessage(event.data);
      } catch (error) {
        console.error("Error handling message:", error);
      }
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      this.updateStatus("disconnected");
      this.reconnect();
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      this.updateStatus("disconnected");
    };
  }

  reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log("Max reconnection attempts reached");
      return;
    }

    this.reconnectAttempts++;
    this.reconnectDelay *= 1.5;
    console.log(
      `Reconnecting in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts})`
    );
    setTimeout(() => this.connect(), this.reconnectDelay);
  }

  handleBinaryMessage(buffer) {
    try {
      const view = new DataView(buffer);
      let offset = 0;

      const numRecords = view.getUint16(offset, true);
      offset += 2;

      if (this.debug) {
        console.log(`Processing ${numRecords} records`);
      }

      for (let i = 0; i < numRecords; i++) {
        const nameLength = view.getUint8(offset);
        offset += 1;
        const stockName = this.decoder.decode(
          new Uint8Array(buffer, offset, nameLength)
        );
        offset += nameLength;

        const instType = view.getUint8(offset);
        offset += 1;

        const lastPrice = view.getFloat32(offset, true);
        offset += 4;
        const strikePrice = view.getFloat32(offset, true);
        offset += 4;
        const lotSize = view.getFloat32(offset, true);
        offset += 4;

        const expiryLength = view.getUint8(offset);
        offset += 1;
        const expiry = this.decoder.decode(
          new Uint8Array(buffer, offset, expiryLength)
        );
        offset += expiryLength;

        const bestBid = view.getFloat32(offset, true);
        offset += 4;
        const bestAsk = view.getFloat32(offset, true);
        offset += 4;

        const lastTradedQuantity = view.getUint32(offset, true);
        offset += 4;
        const averageTradedPrice = view.getFloat32(offset, true);
        offset += 4;
        const volumeTraded = view.getUint32(offset, true);
        offset += 4;
        const totalBuy = view.getUint32(offset, true);
        offset += 4;
        const totalSell = view.getUint32(offset, true);
        offset += 4;
        const high = view.getFloat32(offset, true);
        offset += 4;
        const low = view.getFloat32(offset, true);
        offset += 4;
        const open = view.getFloat32(offset, true);
        offset += 4;
        const close = view.getFloat32(offset, true);
        offset += 4;
        const oi = view.getUint32(offset, true);
        offset += 4;
        const oiHigh = view.getUint32(offset, true);
        offset += 4;
        const oiLow = view.getUint32(offset, true);
        offset += 4;
        const lastTradedTimestamp = new Date(
          view.getUint32(offset, true) * 1000
        );
        offset += 4;
        const exchangeTimestamp = new Date(view.getUint32(offset, true) * 1000);
        offset += 4;

        const instrumentType = ["FUT", "CE", "PE"][instType];

        if (this.debug) {
          console.log(`Record ${i + 1}:`, {
            stockName,
            instrumentType,
            expiry,
            selectedExpiry: this.selectedExpiry,
            lastPrice,
          });
        }

        this.updateData({
          stockName,
          instrumentType,
          lastPrice,
          strikePrice,
          lotSize,
          expiry,
          lastUpdate: new Date(),
          depth: {
            buy: bestBid ? [{ price: bestBid }] : [],
            sell: bestAsk ? [{ price: bestAsk }] : [],
          },
          lastTradedQuantity,
          averageTradedPrice,
          volumeTraded,
          totalBuy,
          totalSell,
          high,
          low,
          open,
          close,
          oi,
          oiHigh,
          oiLow,
          lastTradedTimestamp,
          exchangeTimestamp,
        });
      }
    } catch (error) {
      console.error("Error parsing binary message:", error);
    }
  }

  updateData(data) {
    const key = this.getStockKey(data.stockName, data.expiry);

    if (!this.stockData.has(key)) {
      this.stockData.set(key, {
        stockName: data.stockName,
        future: null,
        futureClose: null,
        strike: null,
        ce: null,
        pe: null,
        lotSize: data.lotSize,
        expiry: data.expiry,
        lastUpdate: data.lastUpdate,
      });
      this.updateExpiryOptions();
      if (data.expiry === this.selectedExpiry) {
        this.createStockRow(key, data.stockName);
        this.sortAndRenderRows();
      }
    }

    const stock = this.stockData.get(key);
    if (!stock) return;

    stock.lastUpdate = data.lastUpdate;

    if (data.instrumentType === "FUT") {
      if (data.lastPrice > 0) {
        stock.future = data.lastPrice;
        stock.futureClose = data.close;
      }
      if (data.lotSize > 0) {
        stock.lotSize = data.lotSize;
      }
    } else {
      if (data.instrumentType === "CE") {
        stock.ce = {
          ...this.getBestBidAsk(data.depth),
          oi: data.oi,
          volumeTraded: data.volumeTraded,
        };
        stock.strike = data.strikePrice;
      } else if (data.instrumentType === "PE") {
        stock.pe = {
          ...this.getBestBidAsk(data.depth),
          oi: data.oi,
          volumeTraded: data.volumeTraded,
        };
        stock.strike = data.strikePrice;
      }
    }

    if (data.expiry === this.selectedExpiry) {
      this.updateRow(key);
    }
  }

  getBestBidAsk(depth) {
    if (!depth || (!depth.buy.length && !depth.sell.length)) {
      return { bid: null, ask: null };
    }

    const bestBid = depth.buy.length > 0 ? depth.buy[0].price : null;
    const bestAsk = depth.sell.length > 0 ? depth.sell[0].price : null;

    return { bid: bestBid, ask: bestAsk };
  }

  getStockKey(stockName, expiry) {
    return `${stockName}-${expiry}`;
  }
}

// Start the application
const app = new MarketDataUI();
