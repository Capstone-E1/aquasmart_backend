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
from datetime import datetime

class AquaSmartSimulator:
    def __init__(self, broker_host="localhost", broker_port=1883, username=None, password=None):
        self.broker_host = broker_host
        self.broker_port = broker_port
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

def main():
    parser = argparse.ArgumentParser(description="AquaSmart MQTT Test Client")
    parser.add_argument("--host", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--username", help="MQTT username")
    parser.add_argument("--password", help="MQTT password")
    parser.add_argument("--mode", choices=["single", "continuous", "scenarios"],
                       default="single", help="Test mode")
    parser.add_argument("--device", default="device_001", help="Device ID for single mode")
    parser.add_argument("--interval", type=int, default=10, help="Interval in seconds for continuous mode")
    parser.add_argument("--duration", type=int, default=300, help="Duration in seconds for continuous mode")

    args = parser.parse_args()

    print("🌊 AquaSmart Water Purification MQTT Test Client")
    print("=" * 60)

    simulator = AquaSmartSimulator(args.host, args.port, args.username, args.password)

    if args.mode == "single":
        print(f"📤 Sending single message from {args.device}")
        simulator.send_single_message(args.device)

    elif args.mode == "continuous":
        simulator.simulate_continuous_data(args.interval, args.duration)

    elif args.mode == "scenarios":
        simulator.send_test_scenarios()

if __name__ == "__main__":
    main()