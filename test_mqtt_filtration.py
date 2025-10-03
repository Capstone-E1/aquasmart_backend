#!/usr/bin/env python3
"""
Enhanced MQTT Test Client for AquaSmart Filtration Progress Testing
This script simulates STM32/ESP8266 device behavior with realistic filtration scenarios.
"""

import json
import time
import random
import threading
import argparse
from datetime import datetime, timedelta
from typing import Dict, Any
import paho.mqtt.client as mqtt

class AquaSmartFiltrationSimulator:
    def __init__(self, broker_host="localhost", broker_port=1883, device_id="test_device_001"):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.device_id = device_id
        self.client = None

        # Simulation state
        self.is_running = False
        self.current_mode = "drinking_water"
        self.filtration_active = False
        self.filtration_start_time = None
        self.target_volume = 50.0
        self.processed_volume = 0.0

        # Sensor simulation parameters
        self.base_flow_rate = 2.5  # L/min
        self.base_ph = 7.2
        self.base_turbidity = 1.2
        self.base_tds = 280.0

        # Topics
        self.sensor_topic = f"aquasmart/sensors/{device_id}/data"
        self.command_topic = "aquasmart/commands/filter"
        self.response_topic = f"aquasmart/commands/{device_id}/response"

    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"✅ Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
            client.subscribe(self.command_topic)
            print(f"📡 Subscribed to command topic: {self.command_topic}")
        else:
            print(f"❌ Failed to connect to MQTT broker, return code {rc}")

    def on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            print(f"📨 Received command: {command}")
            self.handle_filter_command(command)
        except Exception as e:
            print(f"❌ Error processing command: {e}")

    def handle_filter_command(self, command: Dict[str, Any]):
        """Handle filter mode change commands"""
        if command.get("command") == "set_filter_mode":
            new_mode = command.get("mode")

            if new_mode in ["drinking_water", "household_water"]:
                print(f"🔄 Switching from {self.current_mode} to {new_mode}")

                # Simulate mode switching process
                self.publish_command_response("processing", f"Switching to {new_mode} mode")

                # Simulate switching delay
                time.sleep(2)

                self.current_mode = new_mode
                self.start_filtration_process()

                self.publish_command_response("success", f"Successfully switched to {new_mode} mode")
                print(f"✅ Mode changed to {new_mode}")
            else:
                self.publish_command_response("error", f"Invalid mode: {new_mode}")

    def start_filtration_process(self):
        """Start a new filtration process"""
        self.filtration_active = True
        self.filtration_start_time = datetime.now()
        self.processed_volume = 0.0

        # Different target volumes for different modes
        if self.current_mode == "drinking_water":
            self.target_volume = 50.0
        else:
            self.target_volume = 75.0

        print(f"🚰 Starting filtration process: {self.current_mode} mode, target: {self.target_volume}L")

    def publish_command_response(self, status: str, message: str):
        """Publish command response"""
        response = {
            "command": "set_filter_mode",
            "status": status,
            "message": message,
            "timestamp": datetime.now().isoformat()
        }

        self.client.publish(self.response_topic, json.dumps(response))

    def generate_sensor_data(self) -> Dict[str, Any]:
        """Generate realistic sensor data based on current filtration state"""

        # Flow rate varies during filtration
        if self.filtration_active:
            # Simulate flow rate variations during filtration
            elapsed_minutes = (datetime.now() - self.filtration_start_time).total_seconds() / 60
            progress_ratio = self.processed_volume / self.target_volume if self.target_volume > 0 else 0

            # Flow rate starts high and gradually decreases as filters get saturated
            flow_factor = 1.0 - (progress_ratio * 0.3)  # 30% reduction at end
            flow_rate = self.base_flow_rate * flow_factor + random.uniform(-0.2, 0.2)
            flow_rate = max(0.5, flow_rate)  # Minimum flow rate

            # Update processed volume
            self.processed_volume += flow_rate * 0.5  # 0.5 minutes per reading

            # Check if filtration is complete
            if self.processed_volume >= self.target_volume:
                self.filtration_active = False
                self.processed_volume = self.target_volume
                print(f"🎉 Filtration completed! Processed {self.processed_volume}L")
        else:
            # No active filtration, minimal flow
            flow_rate = random.uniform(0.0, 0.1)

        # pH varies based on mode and progress
        if self.current_mode == "drinking_water":
            target_ph = 7.0 + random.uniform(-0.3, 0.3)
        else:
            target_ph = 7.5 + random.uniform(-0.5, 0.5)

        ph = target_ph + random.uniform(-0.1, 0.1)

        # Turbidity and TDS improve during filtration
        if self.filtration_active:
            progress_ratio = self.processed_volume / self.target_volume
            turbidity = self.base_turbidity * (1 - progress_ratio * 0.7) + random.uniform(-0.1, 0.1)
            tds = self.base_tds * (1 - progress_ratio * 0.4) + random.uniform(-10, 10)
        else:
            turbidity = self.base_turbidity + random.uniform(-0.2, 0.2)
            tds = self.base_tds + random.uniform(-15, 15)

        return {
            "flow": max(0, round(flow_rate, 2)),
            "ph": round(ph, 2),
            "turbidity": max(0, round(turbidity, 2)),
            "tds": max(0, round(tds, 1)),
        }

    def publish_sensor_data(self):
        """Publish sensor data with filtration progress info"""
        sensor_data = self.generate_sensor_data()

        # Add filtration metadata for testing
        if self.filtration_active:
            sensor_data["_meta"] = {
                "filtration_active": True,
                "processed_volume": round(self.processed_volume, 2),
                "target_volume": self.target_volume,
                "progress": round((self.processed_volume / self.target_volume) * 100, 1),
                "elapsed_minutes": round((datetime.now() - self.filtration_start_time).total_seconds() / 60, 1)
            }

        payload = json.dumps(sensor_data)
        self.client.publish(self.sensor_topic, payload)

        # Print progress info
        if self.filtration_active:
            progress = (self.processed_volume / self.target_volume) * 100
            print(f"📊 [{datetime.now().strftime('%H:%M:%S')}] "
                  f"Mode: {self.current_mode}, Progress: {progress:.1f}%, "
                  f"Volume: {self.processed_volume:.1f}L/{self.target_volume}L, "
                  f"Flow: {sensor_data['flow']}L/min")
        else:
            print(f"🔬 [{datetime.now().strftime('%H:%M:%S')}] "
                  f"Sensor data: pH={sensor_data['ph']}, Flow={sensor_data['flow']}L/min, "
                  f"Turbidity={sensor_data['turbidity']}NTU, TDS={sensor_data['tds']}ppm")

    def run_simulation(self, duration_minutes=10, interval_seconds=2):
        """Run the simulation for specified duration"""
        print(f"🚀 Starting AquaSmart filtration simulation for {duration_minutes} minutes")
        print(f"📡 Publishing to topic: {self.sensor_topic}")
        print(f"⏱️  Sensor data interval: {interval_seconds} seconds")

        # Setup MQTT client
        self.client = mqtt.Client(client_id=f"aquasmart_simulator_{self.device_id}")
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message

        try:
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()

            self.is_running = True
            start_time = time.time()

            # Start with an initial filtration process
            print(f"\n🔄 Auto-starting initial filtration process in {self.current_mode} mode")
            self.start_filtration_process()

            while self.is_running and (time.time() - start_time) < (duration_minutes * 60):
                self.publish_sensor_data()
                time.sleep(interval_seconds)

            print(f"\n⏹️  Simulation completed after {duration_minutes} minutes")

        except KeyboardInterrupt:
            print(f"\n⏹️  Simulation stopped by user")
        except Exception as e:
            print(f"\n❌ Simulation error: {e}")
        finally:
            self.is_running = False
            if self.client:
                self.client.loop_stop()
                self.client.disconnect()

    def run_scenario_test(self):
        """Run specific test scenarios"""
        print("🧪 Running Filtration Test Scenarios")
        print("=" * 50)

        scenarios = [
            {
                "name": "Quick Drinking Water Cycle",
                "mode": "drinking_water",
                "target_volume": 20.0,
                "duration": 3
            },
            {
                "name": "Household Water Full Cycle",
                "mode": "household_water",
                "target_volume": 40.0,
                "duration": 5
            },
            {
                "name": "High Volume Processing",
                "mode": "drinking_water",
                "target_volume": 100.0,
                "duration": 8
            }
        ]

        # Setup MQTT client
        self.client = mqtt.Client(client_id=f"aquasmart_scenario_test")
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message

        try:
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()

            for i, scenario in enumerate(scenarios):
                print(f"\n📋 Scenario {i+1}: {scenario['name']}")
                print(f"   Mode: {scenario['mode']}")
                print(f"   Target: {scenario['target_volume']}L")
                print(f"   Duration: {scenario['duration']} minutes")

                self.current_mode = scenario['mode']
                self.target_volume = scenario['target_volume']
                self.start_filtration_process()

                # Run scenario
                start_time = time.time()
                while (time.time() - start_time) < (scenario['duration'] * 60):
                    self.publish_sensor_data()

                    if not self.filtration_active:
                        print(f"   ✅ Scenario completed early (filtration finished)")
                        break

                    time.sleep(1)  # Faster updates for scenario testing

                print(f"   📊 Final volume processed: {self.processed_volume:.1f}L")
                time.sleep(2)  # Brief pause between scenarios

        except KeyboardInterrupt:
            print(f"\n⏹️  Scenario testing stopped by user")
        finally:
            if self.client:
                self.client.loop_stop()
                self.client.disconnect()

def main():
    parser = argparse.ArgumentParser(description='AquaSmart Filtration MQTT Simulator')
    parser.add_argument('--host', default='localhost', help='MQTT broker host')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port')
    parser.add_argument('--device', default='test_device_001', help='Device ID')
    parser.add_argument('--duration', type=int, default=10, help='Simulation duration in minutes')
    parser.add_argument('--interval', type=int, default=2, help='Sensor data interval in seconds')
    parser.add_argument('--scenario', action='store_true', help='Run predefined test scenarios')

    args = parser.parse_args()

    simulator = AquaSmartFiltrationSimulator(
        broker_host=args.host,
        broker_port=args.port,
        device_id=args.device
    )

    print("🌊 AquaSmart Filtration MQTT Simulator")
    print("=" * 40)
    print(f"📍 Broker: {args.host}:{args.port}")
    print(f"🔧 Device ID: {args.device}")

    if args.scenario:
        simulator.run_scenario_test()
    else:
        simulator.run_simulation(args.duration, args.interval)

if __name__ == "__main__":
    main()