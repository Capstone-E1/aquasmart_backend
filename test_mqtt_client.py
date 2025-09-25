#!/usr/bin/env python3
"""
MQTT Test Client for AquaSmart Water Purification System
This script simulates STM32/ESP8266 devices sending sensor data
"""

import json
import time
import random
import argparse
import paho.mqtt.client as mqtt
import requests
from datetime import datetime

class AquaSmartSimulator:
    def __init__(self, broker_host="localhost", broker_port=1883, username=None, password=None, api_base_url="http://localhost:8080"):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.api_base_url = api_base_url
        self.client = mqtt.Client()

        if username and password:
            self.client.username_pw_set(username, password)

        self.client.on_connect = self.on_connect
        self.client.on_publish = self.on_publish
        self.client.on_disconnect = self.on_disconnect

        # Device configurations
        self.devices = [
            {"id": "device_001", "location": "Main Tank"},
            {"id": "device_002", "location": "Filter Output"},
            {"id": "device_003", "location": "Storage Tank"}
        ]

    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"✅ Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
        else:
            print(f"❌ Failed to connect to MQTT broker. Return code: {rc}")

    def on_publish(self, client, userdata, mid):
        print(f"📤 Message published (mid: {mid})")

    def on_disconnect(self, client, userdata, rc):
        print(f"🔌 Disconnected from MQTT broker")

    def generate_sensor_data(self):
        """Generate realistic water sensor data"""
        # pH: 6.5-8.5 is good, outside this range indicates issues
        ph = random.uniform(6.0, 9.0)

        # Turbidity: <1 NTU is excellent, 1-4 is good, >4 needs attention
        turbidity = random.uniform(0.1, 5.0)

        # TDS: 100-500 ppm is typically good for drinking water
        tds = random.uniform(50, 600)

        return {
            "ph": round(ph, 2),
            "turbidity": round(turbidity, 2),
            "tds": round(tds, 1)
        }

    def send_sensor_data(self, topic="aquasmart/sensors/data"):
        """Send sensor data to single device system"""
        data = self.generate_sensor_data()
        payload = json.dumps(data)

        result = self.client.publish(topic, payload, qos=1)

        print(f"🌊 Sent sensor data:")
        print(f"   Topic: {topic}")
        print(f"   Data: {payload}")
        print(f"   pH: {data['ph']} ({'Good' if 6.5 <= data['ph'] <= 8.5 else 'Alert!'})")
        print(f"   Turbidity: {data['turbidity']} NTU ({'Good' if data['turbidity'] <= 4 else 'Alert!'})")
        print(f"   TDS: {data['tds']} ppm ({'Good' if data['tds'] <= 500 else 'Alert!'})")
        print()

        return result.is_published()

    def send_single_message(self):
        """Send a single test message"""
        self.client.connect(self.broker_host, self.broker_port, 60)
        self.client.loop_start()
        time.sleep(1)  # Wait for connection

        self.send_sensor_data()

        time.sleep(2)  # Wait for publish
        self.client.loop_stop()
        self.client.disconnect()

    def simulate_continuous_data(self, interval=10, duration=300):
        """Simulate continuous sensor data from all devices"""
        print(f"🚀 Starting continuous simulation for {duration} seconds")
        print(f"📊 Sending data every {interval} seconds from {len(self.devices)} devices")
        print("-" * 60)

        self.client.connect(self.broker_host, self.broker_port, 60)
        self.client.loop_start()
        time.sleep(2)  # Wait for connection

        start_time = time.time()
        message_count = 0

        try:
            while time.time() - start_time < duration:
                for device in self.devices:
                    if self.send_sensor_data(device["id"]):
                        message_count += 1

                print(f"⏰ {datetime.now().strftime('%H:%M:%S')} - Sent {message_count} messages so far")
                time.sleep(interval)

        except KeyboardInterrupt:
            print("\n🛑 Simulation stopped by user")

        finally:
            self.client.loop_stop()
            self.client.disconnect()
            print(f"✨ Simulation complete. Total messages sent: {message_count}")

    def send_test_scenarios(self):
        """Send specific test scenarios"""
        self.client.connect(self.broker_host, self.broker_port, 60)
        self.client.loop_start()
        time.sleep(1)

        scenarios = [
            {"device_id": "device_001", "ph": 7.2, "turbidity": 1.5, "tds": 250.0, "desc": "Normal conditions"},
            {"device_id": "device_002", "ph": 9.1, "turbidity": 0.8, "tds": 180.0, "desc": "High pH - Alkaline"},
            {"device_id": "device_003", "ph": 5.8, "turbidity": 6.2, "tds": 580.0, "desc": "Multiple issues"},
            {"device_id": "device_001", "ph": 7.0, "turbidity": 0.3, "tds": 120.0, "desc": "Excellent quality"},
        ]

        print("🧪 Sending test scenarios:")
        print("-" * 50)

        for i, scenario in enumerate(scenarios, 1):
            topic = f"aquasmart/sensors/{scenario['device_id']}/data"
            data = {
                "device_id": scenario["device_id"],
                "ph": scenario["ph"],
                "turbidity": scenario["turbidity"],
                "tds": scenario["tds"]
            }

            payload = json.dumps(data)
            self.client.publish(topic, payload, qos=1)

            print(f"Test {i}: {scenario['desc']}")
            print(f"  Device: {scenario['device_id']}")
            print(f"  Data: {payload}")
            print()

            time.sleep(3)

        time.sleep(2)
        self.client.loop_stop()
        self.client.disconnect()
        print("✅ Test scenarios complete")

    def test_api_endpoints(self):
        """Test all available API endpoints"""
        print("🔍 Testing API Endpoints")
        print("=" * 60)

        endpoints = [
            {"method": "GET", "path": "/api/v1/health", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/stats", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/sensors/latest", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/sensors/recent", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/sensors/quality", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/devices", "expected_status": 200},
            {"method": "GET", "path": "/api/v1/commands/filter/status", "expected_status": 200},
            {"method": "POST", "path": "/api/v1/commands/filter", "expected_status": 200,
             "data": {"mode": "drinking_water"}},
            {"method": "POST", "path": "/api/v1/commands/filter", "expected_status": 200,
             "data": {"mode": "household_water"}},
        ]

        results = []

        for endpoint in endpoints:
            result = self._test_single_endpoint(endpoint)
            results.append(result)
            time.sleep(0.5)  # Small delay between requests

        # Summary
        print("\n📊 API Test Summary")
        print("-" * 40)
        passed = sum(1 for r in results if r["passed"])
        total = len(results)
        print(f"✅ Passed: {passed}/{total}")
        print(f"❌ Failed: {total - passed}/{total}")

        if passed == total:
            print("🎉 All API endpoints working correctly!")
        else:
            print("⚠️  Some endpoints need attention")

        return results

    def _test_single_endpoint(self, endpoint):
        """Test a single API endpoint"""
        url = f"{self.api_base_url}{endpoint['path']}"
        method = endpoint["method"]
        expected_status = endpoint["expected_status"]

        try:
            if method == "GET":
                response = requests.get(url, timeout=5)
            elif method == "POST":
                headers = {"Content-Type": "application/json"}
                data = json.dumps(endpoint.get("data", {}))
                response = requests.post(url, data=data, headers=headers, timeout=5)
            else:
                print(f"❌ Unsupported method: {method}")
                return {"endpoint": endpoint["path"], "passed": False, "error": "Unsupported method"}

            passed = response.status_code == expected_status

            if passed:
                print(f"✅ {method} {endpoint['path']} - Status: {response.status_code}")
                try:
                    response_data = response.json()
                    if isinstance(response_data, dict) and len(response_data) <= 3:
                        print(f"   Response: {response_data}")
                    else:
                        print(f"   Response: {type(response_data).__name__} with {len(response_data) if hasattr(response_data, '__len__') else 'N/A'} items")
                except:
                    print(f"   Response: {response.text[:100]}...")
            else:
                print(f"❌ {method} {endpoint['path']} - Expected: {expected_status}, Got: {response.status_code}")
                print(f"   Error: {response.text[:200]}")

            return {
                "endpoint": endpoint["path"],
                "method": method,
                "passed": passed,
                "status_code": response.status_code,
                "response_preview": response.text[:100]
            }

        except requests.exceptions.ConnectionError:
            print(f"❌ {method} {endpoint['path']} - Connection failed (Is server running?)")
            return {"endpoint": endpoint["path"], "passed": False, "error": "Connection failed"}
        except requests.exceptions.Timeout:
            print(f"❌ {method} {endpoint['path']} - Request timeout")
            return {"endpoint": endpoint["path"], "passed": False, "error": "Timeout"}
        except Exception as e:
            print(f"❌ {method} {endpoint['path']} - Error: {str(e)}")
            return {"endpoint": endpoint["path"], "passed": False, "error": str(e)}

    def test_water_quality_scenarios(self):
        """Test water quality assessment with different scenarios"""
        print("🧪 Testing Water Quality Assessment")
        print("=" * 60)

        # Connect to MQTT first
        self.client.connect(self.broker_host, self.broker_port, 60)
        self.client.loop_start()
        time.sleep(2)

        quality_scenarios = [
            {
                "name": "Excellent Quality",
                "data": {"flow": 2.5, "ph": 7.0, "turbidity": 0.5, "tds": 200},
                "expected_quality": "Excellent"
            },
            {
                "name": "Good Quality",
                "data": {"flow": 2.0, "ph": 7.5, "turbidity": 1.5, "tds": 400},
                "expected_quality": "Good"
            },
            {
                "name": "Poor Quality (High TDS)",
                "data": {"flow": 1.8, "ph": 7.2, "turbidity": 0.8, "tds": 950},
                "expected_quality": "Poor"
            },
            {
                "name": "Danger Quality (Bad pH)",
                "data": {"flow": 2.2, "ph": 5.5, "turbidity": 1.0, "tds": 300},
                "expected_quality": "Danger"
            },
            {
                "name": "Poor Quality (High Turbidity)",
                "data": {"flow": 2.0, "ph": 7.0, "turbidity": 5.0, "tds": 300},
                "expected_quality": "Poor"
            }
        ]

        for i, scenario in enumerate(quality_scenarios, 1):
            print(f"\nTest {i}: {scenario['name']}")
            print("-" * 30)

            # Send MQTT data
            payload = json.dumps(scenario["data"])
            self.client.publish("aquasmart/sensors/data", payload, qos=1)
            print(f"📤 Sent: {payload}")

            # Wait for processing
            time.sleep(3)

            # Check API response
            try:
                response = requests.get(f"{self.api_base_url}/api/v1/sensors/quality", timeout=5)
                if response.status_code == 200:
                    quality_data = response.json()
                    if quality_data.get("status"):
                        actual_quality = quality_data["status"].get("overall_quality", "Unknown")
                        print(f"🔍 API Response - Overall Quality: {actual_quality}")

                        if actual_quality == scenario["expected_quality"]:
                            print(f"✅ Quality assessment correct!")
                        else:
                            print(f"⚠️  Expected: {scenario['expected_quality']}, Got: {actual_quality}")
                    else:
                        print("⚠️  No quality status in response")
                else:
                    print(f"❌ API request failed: {response.status_code}")
            except Exception as e:
                print(f"❌ Error checking API: {e}")

        self.client.loop_stop()
        self.client.disconnect()
        print("\n✅ Water quality testing complete")

    def run_comprehensive_test(self, include_continuous=False):
        """Run comprehensive test suite"""
        print("🚀 AquaSmart Comprehensive Test Suite")
        print("=" * 60)

        # Test 1: API Endpoints
        print("\n1️⃣ Testing API Endpoints...")
        api_results = self.test_api_endpoints()

        # Test 2: Basic MQTT functionality
        print("\n2️⃣ Testing Basic MQTT Communication...")
        self.send_single_message()

        # Test 3: Water Quality Scenarios
        print("\n3️⃣ Testing Water Quality Assessment...")
        self.test_water_quality_scenarios()

        # Test 4: Filter Commands
        print("\n4️⃣ Testing Filter Commands...")
        self._test_filter_commands()

        # Test 5: Error Handling
        print("\n5️⃣ Testing Error Handling...")
        self._test_error_scenarios()

        if include_continuous:
            print("\n6️⃣ Running Continuous Data Test (30 seconds)...")
            self.simulate_continuous_data(interval=5, duration=30)

        print("\n🎯 Comprehensive Test Complete!")
        return api_results

    def _test_filter_commands(self):
        """Test filter control commands"""
        commands = [
            {"mode": "drinking_water", "desc": "Drinking Water Mode"},
            {"mode": "household_water", "desc": "Household Water Mode"},
            {"mode": "invalid_mode", "desc": "Invalid Mode (should fail)"}
        ]

        for cmd in commands:
            try:
                url = f"{self.api_base_url}/api/v1/commands/filter"
                headers = {"Content-Type": "application/json"}
                data = json.dumps({"mode": cmd["mode"]})

                response = requests.post(url, data=data, headers=headers, timeout=5)

                if cmd["mode"] == "invalid_mode":
                    if response.status_code != 200:
                        print(f"✅ {cmd['desc']} - Correctly rejected (Status: {response.status_code})")
                    else:
                        print(f"⚠️  {cmd['desc']} - Should have been rejected")
                else:
                    if response.status_code == 200:
                        print(f"✅ {cmd['desc']} - Success")
                    else:
                        print(f"❌ {cmd['desc']} - Failed (Status: {response.status_code})")

            except Exception as e:
                print(f"❌ {cmd['desc']} - Error: {e}")

    def _test_error_scenarios(self):
        """Test error handling scenarios"""
        # Test invalid endpoints
        invalid_endpoints = [
            "/api/v1/invalid-endpoint",
            "/api/v1/sensors/nonexistent",
            "/nonexistent"
        ]

        for endpoint in invalid_endpoints:
            try:
                response = requests.get(f"{self.api_base_url}{endpoint}", timeout=5)
                if response.status_code == 404:
                    print(f"✅ Invalid endpoint {endpoint} - Correctly returned 404")
                else:
                    print(f"⚠️  Invalid endpoint {endpoint} - Status: {response.status_code}")
            except Exception as e:
                print(f"❌ Error testing {endpoint}: {e}")

def main():
    parser = argparse.ArgumentParser(description="AquaSmart MQTT Test Client")
    parser.add_argument("--host", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--username", help="MQTT username")
    parser.add_argument("--password", help="MQTT password")
    parser.add_argument("--mode", choices=["single", "continuous", "scenarios", "api", "quality", "comprehensive"],
                       default="single", help="Test mode")
    parser.add_argument("--device", default="device_001", help="Device ID for single mode")
    parser.add_argument("--interval", type=int, default=10, help="Interval in seconds for continuous mode")
    parser.add_argument("--duration", type=int, default=300, help="Duration in seconds for continuous mode")
    parser.add_argument("--api-url", default="http://localhost:8080", help="API base URL")

    args = parser.parse_args()

    print("🌊 AquaSmart Water Purification Test Client")
    print("=" * 60)

    simulator = AquaSmartSimulator(args.host, args.port, args.username, args.password, args.api_url)

    if args.mode == "single":
        print(f"📤 Sending single message")
        simulator.send_single_message()

    elif args.mode == "continuous":
        simulator.simulate_continuous_data(args.interval, args.duration)

    elif args.mode == "scenarios":
        simulator.send_test_scenarios()

    elif args.mode == "api":
        print("🔍 Testing API endpoints only")
        simulator.test_api_endpoints()

    elif args.mode == "quality":
        print("🧪 Testing water quality assessment")
        simulator.test_water_quality_scenarios()

    elif args.mode == "comprehensive":
        print("🚀 Running comprehensive test suite")
        include_continuous = input("Include continuous testing? (y/N): ").lower().startswith('y')
        simulator.run_comprehensive_test(include_continuous)

if __name__ == "__main__":
    main()