import time
import requests

# Define the target address
URL = "http://localhost:8000"

print(f"Starting to send 10 HTTP requests to {URL}...\n")

amount = 1000
time_between = 0.01

for i in range(1, amount + 1):
    try:
        # Send the GET request
        response = requests.get(URL, timeout=5)
        
        # Print the results
        print(f"Request #{i}: Status Code {response.status_code}\tBody {response.text}")
        
    except requests.exceptions.RequestException as e:
        print(f"Request #{i} failed: {e}")
        
    # Optional: Add a 0.5-second pause between requests to prevent accidental rate-limiting
    time.sleep(time_between)

print("\nFinished sending all requests.")