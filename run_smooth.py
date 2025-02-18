import pandas as pd
import numpy as np
import plotly.graph_objects as go
from scipy.interpolate import griddata
from scipy.ndimage import gaussian_filter
import clickhouse_connect
import boto3
from botocore.client import Config
import argparse
import requests

ACCESS_ID = 'DO00CG633YYFBY4NJFMW'
SECRET_KEY = 'fESbgRQ/cWPPIR5R7Pa65nCH/QrPYDrsLCkBN1lPXM0'


# ClickHouse connection details
CLICKHOUSE_HOST = "127.0.0.1"
CLICKHOUSE_PORT = 8123
CLICKHOUSE_DB = "default"
parser = argparse.ArgumentParser(description="Name")
parser.add_argument("name", type=str, help="Name")
args = parser.parse_args()


def fetch_expiries(script):
    client = clickhouse_connect.get_client(host=CLICKHOUSE_HOST, port=CLICKHOUSE_PORT, database=CLICKHOUSE_DB)

    query = f"""select distinct expiry from market_derived_options_log where toDate(timestamp)=toDate(today()) and script='{script}' order by expiry limit 2"""
    df = client.query_df(query)
    return df


# Query ClickHouse for IV, Net Delta, and Timestamp for two datasets
def fetch_data(script, expiry,range=90):
    client = clickhouse_connect.get_client(host=CLICKHOUSE_HOST, port=CLICKHOUSE_PORT, database=CLICKHOUSE_DB)

    query = f"""
    select script,expiry, t, delta, net_delta,iv,s,atm from 
    (
    select script,expiry,toUnixTimestamp(time) as t, delta,sum_otm_delta as net_delta,(iv+iv_last+iv_next)/3 as iv,s,time from
    (
    select *,lagInFrame(iv, 1, iv) OVER (PARTITION BY script,expiry ORDER BY s ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING ) AS iv_last,leadInFrame(iv, 1, iv) OVER (PARTITION BY script,expiry ORDER BY s ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING ) AS iv_next
    from
    (
    select script,expiry,least(cedelta,-pedelta) as delta,(-pedelta-cedelta ) as sum_otm_delta,greatest(ceiv,peiv) as iv, toFloat32(strike) as s,time from 
    (
    select last_value(delta) as cedelta,last_value(iv/100) as ceiv, strike,script,expiry,toStartOfMinute(timestamp) as time from
    (select  strike,iv,delta as delta,script,expiry,instrument_type,timestamp from market_greeks_log where script='{script}' and expiry='{expiry}' and instrument_type='CE'  and iv>0 and iv<40000 and toDate(timestamp)=toDate(today())  order by timestamp )
    group by strike,script,expiry,time
    ) as cetable

    join 
    (
    select last_value(delta) as pedelta,last_value(iv/100) as peiv, strike,script,expiry,toStartOfMinute(timestamp) as time from
    (select  strike,iv,delta as delta,script,expiry,instrument_type,timestamp from market_greeks_log where script='{script}' and expiry='{expiry}' and instrument_type='PE'  and iv>0 and iv<40000 and toDate(timestamp)=toDate(today())  order by timestamp )
    group by strike,script,expiry,time
    ) as petable

    on petable.script=cetable.script and petable.strike=cetable.strike  and petable.time=cetable.time 

    )
    ) where sum_otm_delta<{range} and sum_otm_delta>-{range} order by t desc,s asc

    ) as ivdeltatable

    join 

    (
    select last_value(atm_strike/100) as atm, time from
    (
    select atm_strike, toStartOfMinute(timestamp) as time from market_derived_options_log where toDate(timestamp)=toDate(today()) and expiry='{expiry}' and script='{script}' order by timestamp
    ) group by time
    ) as atmtable on atmtable.time=ivdeltatable.time
    """

    df = client.query_df(query)
    return df
# Fetch data for two different expiry dates
script_name = args.name  # Replace with the required instrument

exps=fetch_expiries(script_name)["expiry"].to_list()
expiry1 = exps[0]
expiry2 = exps[1]

df1 = fetch_data(script_name, expiry1)
df2 = fetch_data(script_name, expiry2)

def process_data(df, common_time_axis=None):
    """ Preprocess the data: Convert time, normalize, smooth, and prepare for plotting """
    df['time'] = pd.to_datetime(df['t'], unit='s')
    df['time_numeric'] = df['time'].astype(np.int64)
    
    df_sorted = df.sort_values(by="time_numeric").copy()
    unique_times = df_sorted["time_numeric"].unique()
    
    if common_time_axis is None:
        time_mapping = {t: i for i, t in enumerate(unique_times)}
        common_time_axis = np.array(list(time_mapping.keys()))  # Save for second dataset
    
    df["time_continuous"] = df["time_numeric"].map({t: i for i, t in enumerate(common_time_axis)})
    
    num_x, num_y = 50, 50
    df['time_bin'] = pd.cut(df['time_continuous'], bins=num_x, labels=False)
    df['net_delta_bin'] = pd.cut(df['net_delta'], bins=num_y, labels=False)
    df_binned = df.groupby(['time_bin', 'net_delta_bin'])['iv'].mean().reset_index()
    
    time_bins = np.linspace(df['time_continuous'].min(), df['time_continuous'].max(), num_x)
    net_delta_bins = np.linspace(df['net_delta'].min(), df['net_delta'].max(), num_y)

    df_binned['time_continuous'] = df_binned['time_bin'].apply(lambda x: time_bins[int(x)] if pd.notna(x) else np.nan)
    df_binned['net_delta'] = df_binned['net_delta_bin'].apply(lambda x: net_delta_bins[int(x)] if pd.notna(x) else np.nan)
    
    df_binned = df_binned.dropna()
    Xi, Yi = np.meshgrid(time_bins, net_delta_bins)
    Zi = griddata((df_binned['time_continuous'], df_binned['net_delta']), df_binned['iv'], (Xi, Yi), method='nearest')
    Zi_smooth = gaussian_filter(Zi, sigma=3)
    
    index_name = df["script"].unique()[0]
    expiry_date = df["expiry"].unique()[0]
    ts_start = df["time_numeric"].min()
    ts_end = df["time_numeric"].max()

    delta_zero_idx = np.abs(Yi).argmin(axis=0)
    time_at_delta_zero = Xi[delta_zero_idx, np.arange(Xi.shape[1])]
    iv_at_delta_zero = Zi_smooth[delta_zero_idx, np.arange(Zi_smooth.shape[1])]
    min_iv_idx = np.nanargmin(Zi_smooth, axis=0)  # Find index where IV is lowest for each time step
    time_at_min_iv = Xi[min_iv_idx, np.arange(Xi.shape[1])]
    delta_at_min_iv = Yi[min_iv_idx, np.arange(Yi.shape[1])]
    iv_at_min_iv = Zi_smooth[min_iv_idx, np.arange(Zi_smooth.shape[1])]

    return Xi, Yi, Zi_smooth, time_at_delta_zero,time_at_min_iv, iv_at_delta_zero, iv_at_min_iv,delta_at_min_iv, index_name, expiry_date, ts_start, ts_end, common_time_axis

# Process first dataset and get a common time axis
Xi1, Yi1, Zi1_smooth, time_at_delta_zero1,time_at_min_iv1, iv_at_delta_zero1, iv_at_min_iv1,delta_at_min_iv1,index1, expiry1, ts_start1, ts_end1, common_time_axis = process_data(df1)

# Process second dataset with the same time axis
Xi2, Yi2, Zi2_smooth, time_at_delta_zero2,time_at_min_iv2, iv_at_delta_zero2, iv_at_min_iv2,delta_at_min_iv2,index2, expiry2, ts_start2, ts_end2, _ = process_data(df2, common_time_axis=common_time_axis)

# Convert timestamps to IST for title
ts_start1_ist = pd.to_datetime(ts_start1, unit="ns").tz_localize("UTC").tz_convert("Asia/Kolkata")
ts_end1_ist = pd.to_datetime(ts_end1, unit="ns").tz_localize("UTC").tz_convert("Asia/Kolkata")
ts_start2_ist = pd.to_datetime(ts_start2, unit="ns").tz_localize("UTC").tz_convert("Asia/Kolkata")
ts_end2_ist = pd.to_datetime(ts_end2, unit="ns").tz_localize("UTC").tz_convert("Asia/Kolkata")

# Define color themes
color_theme_1 = 'Cividis'
color_theme_2 = 'Turbo'

# Create the 3D surface plot
fig = go.Figure()

# Add first dataset's vol surface
fig.add_trace(go.Surface(
    z=Zi1_smooth, x=Xi1, y=Yi1, colorscale=color_theme_1, opacity=0.8,
    name=f"{index1} Exp: {expiry1}", colorbar=dict(x=-0.2, title=f"{index1} IV")
))

# Add second dataset's vol surface
fig.add_trace(go.Surface(
    z=Zi2_smooth, x=Xi2, y=Yi2, colorscale=color_theme_2, opacity=0.7,
    name=f"{index2} Exp: {expiry2}", colorbar=dict(x=1.2, title=f"{index2} IV")
))

# Add red Delta=0 line for first dataset
fig.add_trace(go.Scatter3d(
    x=time_at_min_iv1,
    y=np.zeros_like(time_at_delta_zero1),
    z=iv_at_min_iv1,
    mode='lines',
    line=dict(color='blue', width=5),
    name=f'{index1} Delta 0 IV'
))



# Add a **blue line for the lowest IV over time**
fig.add_trace(go.Scatter3d(
    x=time_at_min_iv2,
    y=delta_at_min_iv2,  # Corresponding delta with lowest IV
    z=iv_at_min_iv2,
    mode='lines',
    line=dict(color='red', width=5),
    name='Min IV Delta Line'
))

# Add a **blue line for the lowest IV over time**
fig.add_trace(go.Scatter3d(
    x=time_at_min_iv1,
    y=delta_at_min_iv1,  # Corresponding delta with lowest IV
    z=iv_at_min_iv1,
    mode='lines',
    line=dict(color='blue', width=5),
    name='Min IV Delta Line'
))

# Add blue Delta=0 line for second dataset
fig.add_trace(go.Scatter3d(
    x=time_at_delta_zero2,
    y=np.zeros_like(time_at_delta_zero2),
    z=iv_at_delta_zero2,
    mode='lines',
    line=dict(color='red', width=5),
    name=f'{index2} Delta 0 IV'
))

# Set title with IST time range
fig.update_layout(
    title=f'3D Vol Surface Over Time for {index1} (Exp: {expiry1}) & {index2} (Exp: {expiry2})<br>'
          f'Time Range (IST): {ts_start1_ist.strftime("%Y-%m-%d %H:%M")} - {ts_end1_ist.strftime("%Y-%m-%d %H:%M")} | '
          f'{ts_start2_ist.strftime("%Y-%m-%d %H:%M")} - {ts_end2_ist.strftime("%Y-%m-%d %H:%M")}',
    scene=dict(
        xaxis_title='Time Elapsed (Continuous Minutes)',
        yaxis_title='Net Delta',
        zaxis_title='Implied Volatility'
    ),
    template="plotly_dark",
    width=1400,  # Increase plot width
    height=1000,  # Increase plot height
    legend=dict(
        x=0.5, y=-0.4,
        xanchor="center", yanchor="top",
        orientation="h", bgcolor="rgba(0,0,0,0)"
    )
)

# Keep only expiry & timestamp range in annotations (remove time range from legend)
fig.update_layout(
    annotations=[
        dict(
            text=f"",
            x=0.5, y=-0.45, showarrow=False, xref="paper", yref="paper",
            font=dict(size=12, color="white")
        )
    ]
)

# Show plot

# Define custom inline CSS
custom_css = """
<style>
    body { background-color: black; color: white; font-family: Arial, sans-serif; }
    .plot-container { width: 100%; height: 100%; }
    .modebar { display: none; }  /* Hides the toolbar */
</style>
"""

# Save the figure as an interactive HTML file with injected CSS
html_path = f'web/{index1}.html'
with open(html_path, "w") as f:
    f.write(custom_css + fig.to_html(full_html=False, include_plotlyjs="cdn"))

# Initiate session
client = boto3.client('s3',
                        region_name='blr1',
                        endpoint_url='https://tradingalgo.blr1.digitaloceanspaces.com',
                        aws_access_key_id=ACCESS_ID,
                        aws_secret_access_key=SECRET_KEY)

# File details
bucket_name = 'vol_surface'
file_path = html_path  # Path to your local HTML file
file_key = file_path.split("/")[-1]  # Extract filename for S3 key

# Upload file with correct Content-Type
client.upload_file(file_path, bucket_name, file_key, ExtraArgs={'ContentType': 'text/html'})
client.put_object_acl( ACL='public-read', Bucket='vol_surface', Key=file_key)



# Generate public URL
html_url = f"https://tradingalgo.blr1.digitaloceanspaces.com/{bucket_name}/{file_key}"

# **Purge Cache (Optional, if using a CDN or caching service)**
purge_url = html_url
try:
    response = requests.request("PURGE", purge_url)
    if response.status_code == 200:
        print(f"Cache purged successfully for {purge_url}")
    else:
        print(f"Cache purge failed: {response.status_code}")
except Exception as e:
    print(f"Error purging cache: {e}")

# Print final URL
print(f"HTML File URL: {html_url}")