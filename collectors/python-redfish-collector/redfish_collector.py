import requests
import structlog
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
log = structlog.get_logger()


class RedfishCollector:
    def __init__(self, device):
        self.device_id = device['id']
        self.hostname = device['hostname']
        self.bmc_ip = device['bmc_ip']
        self.username = device['username']
        self.password = device['password']
        self.port = device.get('port', 443)
        self.scheme = device.get('scheme', 'https')
        self.base_url = f"{self.scheme}://{self.bmc_ip}:{self.port}"
        self.session = requests.Session()
        self.session.verify = False
        self.session.auth = (self.username, self.password)
        self.session.headers.update({'Content-Type': 'application/json'})
        self.vendor = None

    def detect_vendor(self):
        """Auto-detect vendor from Redfish root"""
        try:
            resp = self.session.get(
                f"{self.base_url}/redfish/v1/",
                timeout=15
            )
            resp.raise_for_status()
            data = resp.json()

            vendor = data.get('Vendor', '')
            oem = data.get('Oem', {})

            if 'Dell' in str(oem) or 'Dell' in vendor:
                self.vendor = 'dell'
            elif 'HPE' in str(oem) or 'HPE' in vendor or 'HP' in vendor:
                self.vendor = 'hpe'
            elif 'Lenovo' in str(oem) or 'Lenovo' in vendor:
                self.vendor = 'lenovo'
            elif 'Supermicro' in str(oem) or 'Supermicro' in vendor:
                self.vendor = 'supermicro'
            elif 'Huawei' in str(oem) or 'Huawei' in vendor:
                self.vendor = 'huawei'
            elif 'Cisco' in str(oem) or 'Cisco' in vendor:
                self.vendor = 'cisco'
            else:
                self.vendor = 'generic'

            log.info("vendor_detected",
                     device_id=self.device_id,
                     hostname=self.hostname,
                     vendor=self.vendor)

        except Exception as e:
            log.error("vendor_detection_failed",
                      hostname=self.hostname, error=str(e))
            self.vendor = 'generic'

    def get_chassis_list(self):
        """Get list of chassis"""
        try:
            resp = self.session.get(
                f"{self.base_url}/redfish/v1/Chassis",
                timeout=15
            )
            resp.raise_for_status()
            return resp.json().get('Members', [])
        except Exception as e:
            log.error("chassis_list_failed",
                      hostname=self.hostname, error=str(e))
            return []

    def collect_thermal(self, chassis_url):
        """Collect temperature and fan data"""
        metrics = []
        try:
            resp = self.session.get(
                f"{self.base_url}{chassis_url}/Thermal",
                timeout=15
            )
            resp.raise_for_status()
            data = resp.json()

            for temp in data.get('Temperatures', []):
                reading = temp.get('ReadingCelsius')
                if reading is None:
                    continue
                metrics.append({
                    'name': 'infrasense_redfish_temperature_celsius',
                    'value': float(reading),
                    'labels': {
                        'device_id': self.device_id,
                        'hostname': self.hostname,
                        'sensor_name': temp.get('Name', 'unknown'),
                        'vendor': self.vendor or 'unknown'
                    }
                })

            for fan in data.get('Fans', []):
                reading = fan.get('Reading')
                if reading is None:
                    continue
                metrics.append({
                    'name': 'infrasense_redfish_fan_speed_rpm',
                    'value': float(reading),
                    'labels': {
                        'device_id': self.device_id,
                        'hostname': self.hostname,
                        'fan_name': fan.get('Name', 'unknown'),
                        'vendor': self.vendor or 'unknown'
                    }
                })

        except Exception as e:
            log.error("thermal_collection_failed",
                      hostname=self.hostname, error=str(e))

        return metrics

    def collect_power(self, chassis_url):
        """Collect PSU and power data"""
        metrics = []
        try:
            resp = self.session.get(
                f"{self.base_url}{chassis_url}/Power",
                timeout=15
            )
            resp.raise_for_status()
            data = resp.json()

            for psu in data.get('PowerSupplies', []):
                name = psu.get('Name', 'PSU')
                status = psu.get('Status', {})
                health = status.get('Health', 'Unknown')

                metrics.append({
                    'name': 'infrasense_redfish_psu_status',
                    'value': 1 if health == 'OK' else 0,
                    'labels': {
                        'device_id': self.device_id,
                        'hostname': self.hostname,
                        'psu_name': name,
                        'health': health
                    }
                })

                power_watts = psu.get('PowerOutputWatts') or psu.get('LastPowerOutputWatts')
                if power_watts is not None:
                    metrics.append({
                        'name': 'infrasense_redfish_psu_power_watts',
                        'value': float(power_watts),
                        'labels': {
                            'device_id': self.device_id,
                            'hostname': self.hostname,
                            'psu_name': name
                        }
                    })

        except Exception as e:
            log.error("power_collection_failed",
                      hostname=self.hostname, error=str(e))

        return metrics

    def collect_system_health(self):
        """Collect overall system health"""
        metrics = []
        try:
            resp = self.session.get(
                f"{self.base_url}/redfish/v1/Systems",
                timeout=15
            )
            resp.raise_for_status()
            members = resp.json().get('Members', [])

            for member in members:
                sys_resp = self.session.get(
                    f"{self.base_url}{member['@odata.id']}",
                    timeout=15
                )
                sys_resp.raise_for_status()
                sys_data = sys_resp.json()

                health = sys_data.get('Status', {}).get('Health', 'Unknown')
                metrics.append({
                    'name': 'infrasense_redfish_system_health',
                    'value': 1 if health == 'OK' else 0,
                    'labels': {
                        'device_id': self.device_id,
                        'hostname': self.hostname,
                        'health': health,
                        'model': sys_data.get('Model', 'unknown'),
                        'vendor': self.vendor or 'unknown'
                    }
                })

        except Exception as e:
            log.error("system_health_failed",
                      hostname=self.hostname, error=str(e))

        return metrics

    def collect_event_log(self):
        """Collect Redfish Event Log - vendor specific"""
        events = []

        if self.vendor == 'dell':
            endpoints = [
                '/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Sel/Entries',
                '/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/FaultList/Entries',
            ]
        elif self.vendor == 'hpe':
            endpoints = [
                '/redfish/v1/Systems/1/LogServices/IML/Entries',
                '/redfish/v1/Managers/1/LogServices/IEL/Entries',
            ]
        else:
            endpoints = [
                '/redfish/v1/Systems/1/LogServices/Log1/Entries',
                '/redfish/v1/Managers/1/LogServices/Log1/Entries',
            ]

        for endpoint in endpoints:
            try:
                resp = self.session.get(
                    f"{self.base_url}{endpoint}",
                    timeout=15
                )
                if resp.status_code == 200:
                    data = resp.json()
                    for entry in data.get('Members', []):
                        severity_raw = entry.get('Severity', 'OK').lower()
                        severity = ('critical' if severity_raw == 'critical' else
                                    'warning' if severity_raw == 'warning' else 'info')
                        events.append({
                            'id': entry.get('Id', ''),
                            'severity': severity,
                            'message': entry.get('Message', ''),
                            'type': entry.get('EntryType', 'system'),
                            'component': 'system',
                            'timestamp': entry.get('Created')
                        })
                    break
            except Exception:
                continue

        return events

    def collect_all(self):
        """Collect all metrics from device"""
        all_metrics = []

        self.detect_vendor()

        chassis_list = self.get_chassis_list()
        for chassis in chassis_list:
            chassis_url = chassis.get('@odata.id', '')
            if not chassis_url:
                continue
            all_metrics.extend(self.collect_thermal(chassis_url))
            all_metrics.extend(self.collect_power(chassis_url))

        all_metrics.extend(self.collect_system_health())
        events = self.collect_event_log()

        return all_metrics, events
