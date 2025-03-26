import streamlit as st
import requests
import pandas as pd
import datetime

# Page configuration for a wide layout and trader-like feel
st.set_page_config(
    page_title="Trading Strategy Backtester",
    page_icon="📈",
    layout="wide"
)

# Custom CSS for a professional trading platform look
st.markdown("""
<style>
    .reportview-container {
        background: #f0f2f6;
        padding: 10px;
    }
    .sidebar .sidebar-content {
        background: #ffffff;
        padding: 10px;
        border-radius: 5px;
        box-shadow: 0 2px 5px rgba(0,0,0,0.1);
    }
    .stButton>button {
        width: 100%;
        background-color: #1e90ff;
        color: white;
    }
    .stButton>button:hover {
        background-color: #4169e1;
    }
</style>
""", unsafe_allow_html=True)

st.title("📈 Trading Strategy Backtester")
st.markdown("Backtest your trading strategies with live or CSV data across multiple exchanges and advanced algorithms.")

# Fetch exchanges on app start
if 'exchanges' not in st.session_state:
    with st.spinner("Fetching exchanges..."):
        r = requests.get("http://127.0.0.1:8080/exchanges")
        if r.status_code == 200:
            st.session_state.exchanges = r.json().get("exchanges", [])
        else:
            st.error("Error fetching exchanges.")
            st.session_state.exchanges = []

# Tabs reduced to three
tab1, tab3, tab4 = st.tabs(["Trading Strategy", "Documentation", "About"])

# TAB 1: Trading Strategy
with tab1:
    col1, col2 = st.columns([1, 3])
    with col1:
        st.sidebar.header("💼 User Settings")
        
        # Exchange and Symbol Selection
        with st.sidebar.expander("🌐 Exchange and Symbol", expanded=True):
            selected_exchange = st.selectbox(
                "Select Exchange", 
                st.session_state.exchanges, 
                key="selected_exchange",
                help="Choose the exchange for data retrieval."
            )
            if st.button("Fetch Symbols"):
                with st.spinner(f"Fetching symbols for {selected_exchange}..."):
                    r2 = requests.get(f"http://127.0.0.1:8080/symbols?exchange={selected_exchange}")
                    if r2.status_code == 200:
                        st.session_state.symbols = r2.json().get("symbols", [])
                        st.success(f"Symbols loaded for {selected_exchange}.")
                    else:
                        st.error("Error fetching symbols.")
                        st.session_state.symbols = []
            if 'symbols' in st.session_state and st.session_state.symbols:
                selected_symbol = st.selectbox(
                    "Select Symbol", 
                    st.session_state.symbols, 
                    key="selected_symbol",
                    help="Choose the trading pair to backtest."
                )
            else:
                st.info("Click 'Fetch Symbols' to load available symbols.")
        
        # User Profile
        with st.sidebar.expander("👤 User Profile", expanded=True):
            username = st.text_input("Username", "default_user")
        
        # RSI Parameters
        with st.sidebar.expander("📊 RSI Parameters", expanded=True):
            rsi_period = st.number_input(
                "RSI Period", min_value=2, max_value=50, value=14, step=1,
                help="Number of periods for RSI calculation."
            )
            buy_threshold = st.number_input(
                "Buy/Cover Threshold", min_value=1.0, max_value=99.0, value=30.0, step=1.0,
                help="RSI level to trigger buy (long) or cover (short)."
            )
            sell_threshold = st.number_input(
                "Sell/Short Threshold", min_value=1.0, max_value=99.0, value=70.0, step=1.0,
                help="RSI level to trigger sell (long) or short (short)."
            )
        
        # Moving Average Parameters
        with st.sidebar.expander("📈 Moving Average Parameters", expanded=True):
            ma_period = st.number_input(
                "MA Period", min_value=2, max_value=100, value=20, step=1,
                help="Number of periods for the moving average."
            )
        
        # Trading Parameters
        with st.sidebar.expander("⚙️ Trading Parameters", expanded=True):
            trade_type = st.selectbox(
                "Trade Type", options=["long", "short"],
                help="Simulate long or short trades."
            )
            strategy = st.selectbox(
                "Strategy", 
                options=["RSI", "MA", "RAMSEY", "KAGE", "KITSUNE", "RYU", "SAKURA", "HIKARI", "TENSHI", "ZEN"],
                help="Select a trading strategy:\n"
                     "- RSI: RSI-based signals\n"
                     "- MA: Price crossover of MA\n"
                     "- RSI_MA: Combined RSI and MA\n"
                     "- KAGE: Shadow Logic\n"
                     "- KITSUNE: Fox’s Beam\n"
                     "- RYU: Dragon’s Theory\n"
                     "- SAKURA: Cherry Blossom Mirror\n"
                     "- HIKARI: Advance of Light Momentum\n"
                     "- TENSHI: Angel’s Geometry\n"
                     "- ZEN: Zen Rhythm"
            )
            use_scratch_rsi = st.checkbox(
                "Use Custom RSI", value=False,
                help="Use a custom RSI calculation instead of pandas-ta."
            )
            use_csv = st.checkbox(
                "Use CSV Data", value=False,
                help="Use local CSV minute data instead of live exchange data."
            )
        
        st.sidebar.markdown("---")
        submit_button = st.sidebar.button("Run Backtest", type="primary")
    
    with col2:
        if submit_button:
            with st.spinner("Running backtest..."):
                payload = {
                    "exchange": st.session_state.selected_exchange,
                    "symbol": st.session_state.get("selected_symbol", "BTC/USDT"),
                    "username": username,
                    "rsi_period": rsi_period,
                    "buy_threshold": buy_threshold,
                    "sell_threshold": sell_threshold,
                    "trade_type": trade_type,
                    "strategy": strategy,
                    "ma_period": ma_period,
                    "use_scratch_rsi": use_scratch_rsi,
                    "use_csv": use_csv
                }
                try:
                    response = requests.post("http://127.0.0.1:8080/trade", json=payload)
                    if response.status_code == 200:
                        result = response.json()
                        trades = result.get("trades", [])
                        historical_data = result.get("data", [])
                        plot_img = result.get("plot", "")
                        summary = result.get("summary", {})

                        st.subheader(f"Backtest Results for {payload['symbol']} on {payload['exchange']}")
                        col1_metric, col2_metric, col3_metric, col4_metric = st.columns(4)
                        with col1_metric:
                            st.metric("Total Trades", summary.get("total_trades", 0))
                        with col2_metric:
                            winning = summary.get('winning_trades', 0)
                            total = summary.get('total_trades', 1)
                            st.metric("Winning Trades", f"{winning} ({int(winning/total*100)}%)")
                        with col3_metric:
                            st.metric("Total Return", f"{summary.get('total_profit_pct', 0):.2f}%")
                        with col4_metric:
                            st.metric("Avg. Trade", f"{summary.get('avg_profit_per_trade', 0):.2f}%")

                        st.subheader("📊 Technical Analysis Plot")
                        if plot_img:
                            st.image("data:image/png;base64," + plot_img, use_column_width=True)
                        else:
                            st.info("No plot available.")

                        st.subheader("📋 Trade Details")
                        if trades:
                            trade_df = pd.DataFrame(trades)
                            # Ensure optional columns exist so that DataFrame selection doesn't error out
                            for col in ['entry_rsi', 'exit_rsi']:
                                if col not in trade_df.columns:
                                    trade_df[col] = ""
                            display_columns = ['symbol', 'trade_type', 'entry_time', 'entry_price',
                                               'entry_rsi', 'exit_time', 'exit_price', 'exit_rsi', 'profit_pct']
                            trade_df_display = trade_df[display_columns].copy()
                            trade_df_display['profit_pct'] = trade_df_display['profit_pct'].apply(lambda x: f"{x:.2f}%")
                            st.dataframe(trade_df_display, use_container_width=True)
                        else:
                            st.info("No trades executed. Adjust parameters.")

                        if trades:
                            csv_data = trade_df.to_csv(index=False)
                            st.download_button(
                                label="Download Trade Data (CSV)",
                                data=csv_data,
                                file_name=f"trades_{payload['symbol'].replace('/', '')}{datetime.datetime.now().strftime('%Y%m%d')}.csv",
                                mime="text/csv"
                            )

                        with st.expander("Historical Price and RSI Data"):
                            if historical_data:
                                hist_df = pd.DataFrame(historical_data)
                                st.dataframe(hist_df)
                            else:
                                st.info("No historical data available.")
                    else:
                        st.error(f"Error: {response.text}")
                except Exception as e:
                    st.error(f"Connection error: {str(e)}")
                    st.info("Ensure the backend server is running on http://127.0.0.1:8080")
        else:
            st.info("👈 Configure your parameters in the sidebar and click 'Run Backtest'")
            st.subheader("Strategy Overview")
            st.markdown("""
            *Original Strategies:*
            - *RSI:* Buy when RSI crosses above buy threshold; sell when it crosses below sell threshold.
            - *MA:* Enter on price crossover above (long) or below (short) the MA.
            - *RSI_MA:* Combines RSI and MA signals.
            
            *Advanced Strategies:*
            - *KAGE:* Rolling volatility for regime change.
            - *KITSUNE:* Pattern matching via price changes.
            - *RYU:* Pseudo fractal/chaos metric.
            - *SAKURA:* Median pivot with regression.
            - *HIKARI:* PCA on returns for momentum.
            - *TENSHI:* Local extrema detection.
            - *ZEN:* Bollinger Bands with normalized phase and momentum.
            """)
            
# TAB 2: Documentation
with tab3:
    st.header("Documentation")
    st.subheader("Overview of Trading Strategies")
    st.markdown("""
    This tool supports a variety of trading strategies across multiple exchanges.
    
    *Traditional Strategies:*
    - *RSI:* Momentum via Relative Strength Index.
    - *MA:* Trend via Simple Moving Average.
    - *RSI_MA:* RSI and MA combined.
    
    *Advanced Strategies:*
    - *KAGE:* Volatility-based regime detection.
    - *KITSUNE:* Price change patterns.
    - *RYU:* Chaos theory proxy.
    - *SAKURA:* Regression on price segments.
    - *HIKARI:* Momentum via PCA.
    - *TENSHI:* Support/resistance via extrema.
    - *ZEN:* Bollinger Bands with phase/momentum analysis.
    
    *Data Options:*
    - Live data from selected exchanges via CCXT.
    - CSV minute data (local file).
    
    *Exchange Settings:*
    - Choose an exchange and symbol in the sidebar.
    """)
    st.subheader("Technologies")
    st.markdown("""
    - *Backend:* Go, CCXT (via ccxt-go), Gin, Gonum/Plot
    - *Frontend:* Streamlit
    - *Architecture:* Modular OOP design in Go.
    """)

# TAB 3: About
with tab4:
    st.header("About")
    st.markdown("""
    A proof-of-concept Trading Strategy Backtester for testing algorithmic strategies.
    
    *Features:*
    - Multiple strategies (traditional and advanced).
    - Multi-exchange support with symbol selection.
    - Configurable indicators and parameters.
    - Detailed trade visualizations and metrics.
    
    *Developer Notes:*
    - Go backend with CCXT integration.
    - Streamlit frontend for interactivity.
    
    © 2025 Trading Strategy Backtester
    """)
