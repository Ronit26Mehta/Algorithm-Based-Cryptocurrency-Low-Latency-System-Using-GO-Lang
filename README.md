# Low-Latency-Based-Algorithm-Based-Cryptocurrency-Trader-System-Using-GO-Lang


---

# Trading Backtester

A trading strategy backtester featuring a Go-based backend and a Python (Streamlit) frontend. This project allows you to test various trading strategies with historical and live data using advanced technical analysis methods.

---
## system design :
   procedure diagram:
   ![alt text](<go-lang-based-backend-based-trader/procedure diagram.png>)
   sequenence diagram:
   ![alt text](<go-lang-based-backend-based-trader/sequence diagram.png>)
   system architecture:
   ![alt text](<go-lang-based-backend-based-trader/system architecture.png>)
   class diagram:
   ![alt text](<go-lang-based-backend-based-trader/class diagram.png>)

## Table of Contents

- [Overview](#overview)
- [Repository Structure](#repository-structure)
- [Requirements](#requirements)
  - [Go Backend](#go-backend)
  - [Python Frontend](#python-frontend)
- [Installation and Setup](#installation-and-setup)
  - [Backend Setup](#backend-setup)
  - [Frontend Setup](#frontend-setup)
- [Running the Application](#running-the-application)
  - [Running the Go Server](#running-the-go-server)
  - [Running the Streamlit Frontend](#running-the-streamlit-frontend)
- [Usage](#usage)
- [Project Details](#project-details)
  - [Backend Code Overview](#backend-code-overview)
  - [Frontend Code Overview](#frontend-code-overview)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

This repository contains two main components:

- **Backend (Go):** Implements the trading logic, data fetching from Binance, multiple strategy implementations (e.g., KAGE, KITSUNE, RYU, SAKURA, HIKARI, TENSHI, ZEN, RAMSEY), and endpoints using the Gin framework.
- **Frontend (Python):** A Streamlit-based UI that enables users to set parameters, run backtests, view results, and download trade data.

---

## Repository Structure

```plaintext
trading-backtester/
├── backend/
│   ├── main.go          # Main server code with strategy implementations
│   └── go.mod           # Go module file for dependency management
├── frontend/
│   ├── frontend.py      # Streamlit application code
│   └── requirements.txt # Python dependencies for the frontend
└── README.md            # Project documentation and setup instructions
```

---

## Requirements

### Go Backend

- **Go:** Version 1.18 or higher
- **Dependencies:**  
  - [github.com/adshao/go-binance/v2](https://github.com/adshao/go-binance)  
  - [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)  
  - [gonum.org/v1/plot](https://github.com/gonum/plot)

A `go.mod` file is provided in the `backend/` directory for dependency management.

### Python Frontend

- **Python:** Version 3.8 or higher
- **Dependencies:**  
  - [streamlit](https://streamlit.io/)
  - [requests](https://docs.python-requests.org/)
  - [pandas](https://pandas.pydata.org/)

The `frontend/requirements.txt` file lists these dependencies.

---

## Installation and Setup

### Backend Setup

1. **Install Go:**  
   Download and install Go from the [official website](https://golang.org/dl/).

2. **Clone the Repository:**  
   ```bash
   git clone https://github.com/yourusername/trading-backtester.git
   cd trading-backtester/backend
   ```

3. **Initialize and Download Dependencies:**  
   Run the following command in the `backend` folder to download required packages:
   ```bash
   go mod tidy
   ```
   This will resolve and download all the necessary dependencies based on your `go.mod` file.

### Frontend Setup

1. **Install Python:**  
   Download and install Python from [python.org](https://www.python.org/downloads/).

2. **Create and Activate a Virtual Environment (Optional but Recommended):**  
   ```bash
   python -m venv venv
   # On macOS/Linux:
   source venv/bin/activate
   # On Windows:
   venv\Scripts\activate
   ```

3. **Install Python Dependencies:**  
   Navigate to the `frontend` folder and install dependencies:
   ```bash
   cd ../frontend
   pip install -r requirements.txt
   ```

---

## Running the Application

### Running the Go Server

1. Navigate to the `backend` folder if not already there:
   ```bash
   cd trading-backtester/backend
   ```
2. Run the server:
   ```bash
   go run main.go
   ```
3. The server will start on port **8080** and expose endpoints such as:
   - `POST /trade`
   - `GET /exchanges`
   - `GET /symbols`

### Running the Streamlit Frontend

1. Open a new terminal window.
2. Navigate to the `frontend` folder:
   ```bash
   cd trading-backtester/frontend
   ```
3. Run the Streamlit application:
   ```bash
   streamlit run frontend.py
   ```
4. A web browser should open automatically. If not, navigate to the provided localhost URL to access the UI.

---

## Usage

- **Backend:**  
  The Go backend handles data fetching (from Binance or a CSV file) and executes a variety of trading strategies based on parameters received via HTTP POST requests to `/trade`.

- **Frontend:**  
  The Streamlit UI allows you to select an exchange, fetch symbols, configure parameters (RSI, moving averages, trade type, etc.), and run a backtest. The UI then displays trade details, metrics, plots, and offers a CSV download option for trade data.

---

## Project Details

### Backend Code Overview

- **Data Types & Utilities:**  
  Defines data structures like `Candle` and `Trade`. Implements utility functions to load CSV data, calculate indicators (e.g., novel stochastic oscillator), generate plots, and log trades.

- **Strategy Implementations:**  
  Contains implementations for several strategies (e.g., KAGE, KITSUNE, RYU, SAKURA, HIKARI, TENSHI, ZEN, RAMSEY). Each strategy processes historical data, generates trade signals, and produces a summary of trades.

- **HTTP Endpoints:**  
  Uses the Gin framework to create endpoints for trading (`/trade`), fetching exchanges (`/exchanges`), and retrieving symbols (`/symbols`).

### Frontend Code Overview

- **UI Components:**  
  Built with Streamlit, the frontend provides interactive tabs for trading strategy setup, documentation, and information about the project.

- **User Interaction:**  
  The sidebar allows users to configure exchange selection, symbol fetch, and various trading parameters. On running a backtest, it sends a request to the backend and displays results including trade metrics and visual plots.

---

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Commit your changes and push your branch.
4. Open a pull request detailing your changes.

---

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.

---
