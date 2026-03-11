import subprocess
import time
import argparse
import os
import sys
from datetime import datetime, timedelta
import xml.etree.ElementTree as ET
import glob
import json
import urllib.request
import urllib.error
import re
import statistics
import math

# Google Sheets API imports (optional)
try:
    from google.oauth2 import service_account
    from googleapiclient.discovery import build
    GOOGLE_SHEETS_AVAILABLE = True
except ImportError:
    GOOGLE_SHEETS_AVAILABLE = False

# SciPy for Fisher's Exact Test (optional)
try:
    from scipy.stats import fisher_exact
    SCIPY_AVAILABLE = True
except ImportError:
    SCIPY_AVAILABLE = False

def log(message: str, logfile: str = None):
    timestamped = f"[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] {message}"
    print(timestamped)
    if logfile:
        with open(logfile, "a") as f:
            f.write(timestamped + "\n")

def send_slack_message(webhook_url: str, message: str, logfile: str = None):
    """
    Send message to Slack channel

    Args:
        webhook_url: Slack Webhook URL
        message: Message content to send
        logfile: Log file path (optional)

    Returns:
        bool: Returns True on success, False on failure
    """
    try:
        payload = {
            "text": message
        }

        data = json.dumps(payload).encode('utf-8')
        req = urllib.request.Request(
            webhook_url,
            data=data,
            headers={'Content-Type': 'application/json'}
        )

        with urllib.request.urlopen(req, timeout=10) as response:
            if response.status == 200:
                log(f"Slack notification sent successfully", logfile)
                return True
            else:
                log(f"Failed to send Slack notification: HTTP {response.status}", logfile)
                return False

    except urllib.error.URLError as e:
        log(f"Failed to send Slack notification: {e}", logfile)
        return False
    except Exception as e:
        log(f"Unexpected error sending Slack notification: {e}", logfile)
        return False

def save_command(cmd: list, path: str = "command.log"):
    """Save executed command to file"""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    line = f"[{timestamp}] {' '.join(cmd)}\n"
    with open(path, "a") as f:
        f.write(line)

def write_to_google_sheet(spreadsheet_id: str, sheet_name: str, credentials_file: str,
                          test_data: dict, logfile: str = None, start_cell: str = None,
                          mark_failed: bool = False):
    """
    Write test results to Google Sheets

    Args:
        spreadsheet_id: The ID of the Google Spreadsheet (from URL)
        sheet_name: Name of the sheet/tab (e.g., "OLM Overall Performance Test Report")
        credentials_file: Path to Google service account credentials JSON file
        test_data: Dictionary containing test results
        logfile: Log file path (optional)
        start_cell: Starting cell for writing data (e.g., 'B30'). If None, auto-detects header.
        mark_failed: If True, mark the row with red background (test failed)

    Returns:
        bool: True if successful, False otherwise
    """
    if not GOOGLE_SHEETS_AVAILABLE:
        log("ERROR: Google Sheets API libraries not available. Install with: pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client", logfile)
        return False

    try:
        # Load credentials
        log(f"Loading Google Sheets credentials from {credentials_file}...", logfile)
        credentials = service_account.Credentials.from_service_account_file(
            credentials_file,
            scopes=['https://www.googleapis.com/auth/spreadsheets']
        )

        # Build the service
        service = build('sheets', 'v4', credentials=credentials)
        sheet = service.spreadsheets()

        # Determine status based on avg_pass_rate vs threshold
        # Status: PASS if avg_pass_rate >= threshold, FAIL otherwise
        status = test_data.get('status', 'UNKNOWN')

        # Prepare the row data
        # Format: [Status, Date, OCP Version, Test Duration, Total Runs, Avg Pass Rate, Min Pass Rate, Max Pass Rate, Total Tests, Total Failures, Skipped, Failed Tests Summary, Notes]
        row_data = [
            status,
            test_data.get('date', datetime.now().strftime('%Y-%m-%d')),
            test_data.get('ocp_version', 'N/A'),
            test_data.get('test_duration', 'N/A'),
            str(test_data.get('total_runs', 0)),
            test_data.get('avg_pass_rate', 0.0),
            test_data.get('min_pass_rate', 0.0),
            test_data.get('max_pass_rate', 0.0),
            str(test_data.get('total_tests', 0)),
            str(test_data.get('total_failures', 0)),
            str(test_data.get('skipped', 0)),
            test_data.get('failed_tests_summary', ''),
            test_data.get('notes', '')
        ]

        body = {
            'values': [row_data]
        }

        # Auto-detect header location by searching for "Status" in the sheet
        if not start_cell:
            log(f"Searching for header 'Status' in sheet '{sheet_name}'...", logfile)
            # Read all data from the sheet to find the header
            result = sheet.values().get(
                spreadsheetId=spreadsheet_id,
                range=f"{sheet_name}"
            ).execute()
            values = result.get('values', [])

            header_row = None
            header_col = None
            for row_idx, row in enumerate(values):
                for col_idx, cell in enumerate(row):
                    if cell and str(cell).strip().lower() == 'status':
                        header_row = row_idx + 1  # Convert to 1-based
                        header_col = col_idx  # 0-based for chr() calculation
                        break
                if header_row:
                    break

            if header_row and header_col is not None:
                # Find the next empty row after the header
                start_col_letter = chr(ord('A') + header_col)
                data_start_row = header_row + 1

                # Check existing data rows to find the next empty row
                for row_idx in range(header_row, len(values)):
                    row = values[row_idx] if row_idx < len(values) else []
                    # Check if the Status column (first column of our data) is empty
                    if header_col >= len(row) or not row[header_col] or str(row[header_col]).strip() == '':
                        data_start_row = row_idx + 1  # Convert to 1-based
                        break
                    data_start_row = row_idx + 2  # Next row after current

                start_cell = f"{start_col_letter}{data_start_row}"
                log(f"Found header at row {header_row}, will write data at {start_cell}", logfile)
            else:
                log(f"Header 'Status' not found in sheet, will append to end", logfile)

        if start_cell:
            # Write to specific cell location
            start_col = ''.join(filter(str.isalpha, start_cell))
            start_row = ''.join(filter(str.isdigit, start_cell))
            # Calculate end column
            end_col_num = ord(start_col.upper()) - ord('A') + len(row_data)
            end_col = chr(ord('A') + end_col_num - 1)
            range_name = f"{sheet_name}!{start_cell}:{end_col}{start_row}"

            log(f"Writing test results to Google Sheet: {spreadsheet_id}, sheet: {sheet_name}, range: {range_name}", logfile)
            result = sheet.values().update(
                spreadsheetId=spreadsheet_id,
                range=range_name,
                valueInputOption='USER_ENTERED',
                body=body
            ).execute()
            log(f"Successfully wrote {result.get('updatedCells')} cells to Google Sheet at {start_cell}", logfile)

            # Mark row with red background if failed
            if mark_failed:
                row_index = int(start_row) - 1  # Convert to 0-based index
                red_color = {'red': 1.0, 'green': 0.8, 'blue': 0.8}
                set_row_background_color(sheet, spreadsheet_id, sheet_name, row_index, red_color, logfile)
                log(f"Marked row as FAILED (red background)", logfile)
        else:
            # Append to the end of the sheet (fallback)
            range_name = f"{sheet_name}!A:M"  # 13 columns
            log(f"Writing test results to Google Sheet: {spreadsheet_id}, sheet: {sheet_name}", logfile)
            result = sheet.values().append(
                spreadsheetId=spreadsheet_id,
                range=range_name,
                valueInputOption='USER_ENTERED',
                insertDataOption='INSERT_ROWS',
                body=body
            ).execute()
            log(f"Successfully wrote {result.get('updates').get('updatedCells')} cells to Google Sheet", logfile)

            # Mark row with red background if failed
            if mark_failed:
                updated_range = result.get('updates', {}).get('updatedRange', '')
                if updated_range:
                    import re
                    match = re.search(r'[A-Z]+(\d+):', updated_range)
                    if match:
                        row_index = int(match.group(1)) - 1  # Convert to 0-based index
                        red_color = {'red': 1.0, 'green': 0.8, 'blue': 0.8}
                        set_row_background_color(sheet, spreadsheet_id, sheet_name, row_index, red_color, logfile)
                        log(f"Marked row as FAILED (red background)", logfile)
        return True

    except FileNotFoundError:
        log(f"ERROR: Credentials file not found: {credentials_file}", logfile)
        return False
    except Exception as e:
        log(f"ERROR: Failed to write to Google Sheet: {e}", logfile)
        return False

def set_row_background_color(sheet_service, spreadsheet_id: str, sheet_name: str,
                              row_index: int, color: dict, logfile: str = None):
    """
    Set background color for a specific row in Google Sheets

    Args:
        sheet_service: Google Sheets service object
        spreadsheet_id: The ID of the Google Spreadsheet
        sheet_name: Name of the sheet/tab
        row_index: 0-based row index to color
        color: RGB color dict, e.g., {'red': 1.0, 'green': 0.8, 'blue': 0.8} for light red
        logfile: Log file path (optional)

    Returns:
        bool: True if successful, False otherwise
    """
    try:
        # Get the sheet ID from sheet name
        spreadsheet = sheet_service.get(spreadsheetId=spreadsheet_id).execute()
        sheet_id = None
        for s in spreadsheet['sheets']:
            if s['properties']['title'] == sheet_name:
                sheet_id = s['properties']['sheetId']
                break

        if sheet_id is None:
            log(f"Warning: Could not find sheet '{sheet_name}' to set color", logfile)
            return False

        # Create the request to set background color
        request = {
            'requests': [{
                'repeatCell': {
                    'range': {
                        'sheetId': sheet_id,
                        'startRowIndex': row_index,
                        'endRowIndex': row_index + 1,
                        'startColumnIndex': 0,
                        'endColumnIndex': 10  # Columns A-J
                    },
                    'cell': {
                        'userEnteredFormat': {
                            'backgroundColor': color
                        }
                    },
                    'fields': 'userEnteredFormat.backgroundColor'
                }
            }]
        }

        sheet_service.batchUpdate(spreadsheetId=spreadsheet_id, body=request).execute()
        log(f"Set row {row_index + 1} background color in sheet '{sheet_name}'", logfile)
        return True

    except Exception as e:
        log(f"Warning: Failed to set row background color: {e}", logfile)
        return False


def write_per_run_results_to_google_sheet(spreadsheet_id: str, sheet_name: str, credentials_file: str,
                                          run_number: int, date: str, ocp_version: str, pass_rate: float,
                                          total_tests: int, started_time: str, logfile: str = None,
                                          skipped: int = 0, failures: int = 0, errors: int = 0,
                                          failed_tests: list = None, is_anomaly: bool = False):
    """
    Write per-run test results to Google Sheets (after each test run)

    Args:
        spreadsheet_id: The ID of the Google Spreadsheet (from URL)
        sheet_name: Name of the sheet/tab for per-run results (e.g., "Long-duration Test Results")
        credentials_file: Path to Google service account credentials JSON file
        run_number: Current run number
        date: Date of this run (YYYY-MM-DD)
        ocp_version: OCP version
        pass_rate: Pass rate for this run (percentage)
        total_tests: Total number of tests in this run
        started_time: When the test session started (YYYY-MM-DD HH:MM:SS)
        logfile: Log file path (optional)
        skipped: Number of skipped tests (default: 0)
        failures: Number of failed tests (default: 0)
        errors: Number of error tests (default: 0)
        failed_tests: List of failed test details (optional)
        is_anomaly: If True, mark this row with red background (anomaly detected)

    Returns:
        bool: True if successful, False otherwise
    """
    if not GOOGLE_SHEETS_AVAILABLE:
        log("ERROR: Google Sheets API libraries not available. Install with: pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client", logfile)
        return False

    try:
        # Load credentials
        credentials = service_account.Credentials.from_service_account_file(
            credentials_file,
            scopes=['https://www.googleapis.com/auth/spreadsheets']
        )

        # Build the service
        service = build('sheets', 'v4', credentials=credentials)
        sheet = service.spreadsheets()

        # Check if the sheet exists, create if not
        try:
            spreadsheet = sheet.get(spreadsheetId=spreadsheet_id).execute()
            sheet_exists = any(s['properties']['title'] == sheet_name for s in spreadsheet['sheets'])

            if not sheet_exists:
                log(f"Creating new sheet '{sheet_name}' in spreadsheet...", logfile)
                request_body = {
                    'requests': [{
                        'addSheet': {
                            'properties': {
                                'title': sheet_name
                            }
                        }
                    }]
                }
                sheet.batchUpdate(spreadsheetId=spreadsheet_id, body=request_body).execute()

                # Add header row
                header_row = [['Run #', 'Date', 'OCP Version', 'Pass Rate (%)', 'Total Tests', 'Skipped', 'Failures', 'Errors', 'Failed Test Names', 'Started Time']]
                header_range = f"{sheet_name}!A1:J1"
                sheet.values().update(
                    spreadsheetId=spreadsheet_id,
                    range=header_range,
                    valueInputOption='USER_ENTERED',
                    body={'values': header_row}
                ).execute()
                log(f"Created sheet '{sheet_name}' with header row", logfile)

        except Exception as e:
            log(f"Warning: Could not check/create sheet '{sheet_name}': {e}", logfile)

        # Prepare failed test names string (semicolon-separated, limited to avoid cell overflow)
        failed_test_names = ""
        if failed_tests:
            # Extract test names and join with semicolons
            names = [t.get('name', 'Unknown') for t in failed_tests]
            failed_test_names = "; ".join(names)
            # Limit to 5000 characters to avoid Google Sheets cell limit
            if len(failed_test_names) > 5000:
                failed_test_names = failed_test_names[:4997] + "..."

        # Prepare the row data
        # Format: [Run #, Date, OCP Version, Pass Rate (%), Total Tests, Skipped, Failures, Errors, Failed Test Names, Started Time]
        # Note: Convert numeric values to strings to prevent Google Sheets from auto-formatting as dates
        row_data = [
            str(run_number),
            date,
            ocp_version,
            f"{pass_rate:.2f}",
            str(total_tests),
            str(skipped),
            str(failures),
            str(errors),
            failed_test_names,
            started_time
        ]

        # Append the data to the sheet
        range_name = f"{sheet_name}!A:J"
        body = {
            'values': [row_data]
        }

        log(f"Writing run #{run_number} results to Google Sheet '{sheet_name}'...", logfile)
        result = sheet.values().append(
            spreadsheetId=spreadsheet_id,
            range=range_name,
            valueInputOption='USER_ENTERED',
            insertDataOption='INSERT_ROWS',
            body=body
        ).execute()

        log(f"Successfully wrote {result.get('updates').get('updatedCells')} cells (per-run results) to Google Sheet", logfile)

        # If this run is an anomaly, mark the row with red background
        if is_anomaly:
            # Get the row index from the updated range (e.g., "Sheet!A5:J5" -> row 5 -> index 4)
            updated_range = result.get('updates', {}).get('updatedRange', '')
            if updated_range:
                # Extract row number from range like "Sheet!A5:J5"
                import re
                match = re.search(r'[A-Z]+(\d+):', updated_range)
                if match:
                    row_number = int(match.group(1))
                    row_index = row_number - 1  # Convert to 0-based index
                    # Light red color
                    red_color = {'red': 1.0, 'green': 0.8, 'blue': 0.8}
                    set_row_background_color(sheet, spreadsheet_id, sheet_name, row_index, red_color, logfile)
                    log(f"Marked run #{run_number} as ANOMALY (red background)", logfile)

        return True

    except FileNotFoundError:
        log(f"ERROR: Credentials file not found: {credentials_file}", logfile)
        return False
    except Exception as e:
        log(f"ERROR: Failed to write per-run results to Google Sheet: {e}", logfile)
        import traceback
        log(f"Traceback: {traceback.format_exc()}", logfile)
        return False

def write_failure_details_to_google_sheet(spreadsheet_id: str, sheet_name: str, credentials_file: str,
                                          run_number: int, timestamp: str, failed_tests: list, logfile: str = None):
    """
    Write failure details to a separate Google Sheets tab

    Args:
        spreadsheet_id: The ID of the Google Spreadsheet (from URL)
        sheet_name: Name of the sheet/tab for failed tests (e.g., "Failed Tests")
        credentials_file: Path to Google service account credentials JSON file
        run_number: Current run number
        timestamp: Timestamp of the test run
        failed_tests: List of dicts containing failure details (from parse_junit_results with return_details=True)
                     Each dict has: {'name': str, 'type': str, 'message': str}
        logfile: Log file path (optional)

    Returns:
        bool: True if successful, False otherwise
    """
    if not GOOGLE_SHEETS_AVAILABLE:
        log("ERROR: Google Sheets API libraries not available. Install with: pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client", logfile)
        return False

    if not failed_tests:
        log(f"No failed tests to write to Google Sheets for run #{run_number}", logfile)
        return True

    try:
        # Load credentials
        credentials = service_account.Credentials.from_service_account_file(
            credentials_file,
            scopes=['https://www.googleapis.com/auth/spreadsheets']
        )

        # Build the service
        service = build('sheets', 'v4', credentials=credentials)
        sheet = service.spreadsheets()

        # Check if the "Failed Tests" sheet exists, create if not
        try:
            spreadsheet = sheet.get(spreadsheetId=spreadsheet_id).execute()
            sheet_exists = any(s['properties']['title'] == sheet_name for s in spreadsheet['sheets'])

            if not sheet_exists:
                log(f"Creating new sheet '{sheet_name}' in spreadsheet...", logfile)
                request_body = {
                    'requests': [{
                        'addSheet': {
                            'properties': {
                                'title': sheet_name
                            }
                        }
                    }]
                }
                sheet.batchUpdate(spreadsheetId=spreadsheet_id, body=request_body).execute()

                # Add header row
                header_row = [['Run #', 'Timestamp', 'Test Name', 'Failure Type', 'Error Message']]
                header_range = f"{sheet_name}!A1:E1"
                sheet.values().update(
                    spreadsheetId=spreadsheet_id,
                    range=header_range,
                    valueInputOption='USER_ENTERED',
                    body={'values': header_row}
                ).execute()
                log(f"Created sheet '{sheet_name}' with header row", logfile)

        except Exception as e:
            log(f"Warning: Could not check/create sheet '{sheet_name}': {e}", logfile)

        # Prepare failure data rows
        # Format: [Run #, Timestamp, Test Name, Failure Type, Error Message]
        failure_rows = []
        for failed_test in failed_tests:
            failure_rows.append([
                run_number,
                timestamp,
                failed_test.get('name', 'Unknown'),
                failed_test.get('type', 'unknown').upper(),
                failed_test.get('message', 'No message')[:500]  # Limit message to 500 chars
            ])

        # Append the failure data to the sheet
        range_name = f"{sheet_name}!A:E"
        body = {
            'values': failure_rows
        }

        log(f"Writing {len(failure_rows)} failed test(s) to Google Sheet '{sheet_name}' for run #{run_number}...", logfile)
        result = sheet.values().append(
            spreadsheetId=spreadsheet_id,
            range=range_name,
            valueInputOption='USER_ENTERED',
            insertDataOption='INSERT_ROWS',
            body=body
        ).execute()

        log(f"Successfully wrote {result.get('updates').get('updatedCells')} cells (failures) to Google Sheet", logfile)
        return True

    except FileNotFoundError:
        log(f"ERROR: Credentials file not found: {credentials_file}", logfile)
        return False
    except Exception as e:
        log(f"ERROR: Failed to write failure details to Google Sheet: {e}", logfile)
        import traceback
        log(f"Traceback: {traceback.format_exc()}", logfile)
        return False

def run_must_gather(output_dir: str = "./must-gather", logfile: str = None):
    """
    Execute 'oc adm must-gather' to collect cluster diagnostic information

    Args:
        output_dir: Directory to store must-gather output
        logfile: Log file path (optional)

    Returns:
        bool: True if successful, False otherwise
    """
    try:
        # Create output directory if it doesn't exist
        os.makedirs(output_dir, exist_ok=True)

        # Generate timestamped directory name
        timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
        dest_dir = os.path.join(output_dir, f"must-gather-{timestamp}")

        log("=" * 60, logfile)
        log("Executing 'oc adm must-gather' to collect cluster diagnostics...", logfile)
        log(f"Output directory: {dest_dir}", logfile)
        log("=" * 60, logfile)
        log("NOTE: This may take 5-10 minutes to complete...", logfile)

        # Execute must-gather command
        cmd = ["oc", "adm", "must-gather", f"--dest-dir={dest_dir}"]

        log(f"Running: {' '.join(cmd)}", logfile)

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=900  # 15 minutes timeout
        )

        if result.returncode == 0:
            log("=" * 60, logfile)
            log(f"SUCCESS: must-gather completed successfully", logfile)
            log(f"Output saved to: {dest_dir}", logfile)
            log("=" * 60, logfile)
            if result.stdout:
                log(f"Output:\n{result.stdout}", logfile)
            return True
        else:
            log("!" * 60, logfile)
            log(f"ERROR: must-gather failed with return code {result.returncode}", logfile)
            log("!" * 60, logfile)
            if result.stderr:
                log(f"Error output:\n{result.stderr}", logfile)
            if result.stdout:
                log(f"Standard output:\n{result.stdout}", logfile)
            return False

    except subprocess.TimeoutExpired:
        log("!" * 60, logfile)
        log("ERROR: must-gather timed out after 15 minutes", logfile)
        log("!" * 60, logfile)
        return False
    except Exception as e:
        log("!" * 60, logfile)
        log(f"ERROR: Failed to execute must-gather: {e}", logfile)
        log("!" * 60, logfile)
        import traceback
        log(f"Traceback: {traceback.format_exc()}", logfile)
        return False

def get_ocp_version(logfile: str = None):
    """
    Get the OCP cluster version

    Returns:
        str: OCP version or 'Unknown'
    """
    try:
        result = subprocess.run(
            ["oc", "version", "-o", "json"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            version_data = json.loads(result.stdout)
            # Get the OpenShift version
            ocp_version = version_data.get('openshiftVersion', 'Unknown')
            return ocp_version
        else:
            log(f"Failed to get OCP version: {result.stderr}", logfile)
            return "Unknown"
    except Exception as e:
        log(f"Error getting OCP version: {e}", logfile)
        return "Unknown"

def save_pass_rates(pass_rates: list, filename: str = "pass_rates_history.json"):
    """
    Save pass rates history to a JSON file

    Args:
        pass_rates: List of pass rate values
        filename: Filename to save to
    """
    try:
        data = {
            'timestamp': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'pass_rates': pass_rates
        }
        with open(filename, 'w') as f:
            json.dump(data, f, indent=2)
    except Exception as e:
        log(f"Warning: Failed to save pass rates history: {e}")

def load_pass_rates(filename: str = "pass_rates_history.json"):
    """
    Load pass rates history from a JSON file

    Args:
        filename: Filename to load from

    Returns:
        list: List of pass rates, or empty list if file doesn't exist
    """
    try:
        if os.path.exists(filename):
            with open(filename, 'r') as f:
                data = json.load(f)
                return data.get('pass_rates', [])
        return []
    except Exception as e:
        log(f"Warning: Failed to load pass rates history: {e}")
        return []

def save_test_case_history(test_case_results: dict, filename: str = "test_case_history.json"):
    """
    Save test case results history to a JSON file

    Args:
        test_case_results: Dict mapping test names to list of results (True/False)
                          Example: {"test1": [True, False, True], "test2": [True, True, False]}
        filename: Filename to save to
    """
    try:
        data = {
            'timestamp': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'test_cases': test_case_results
        }
        with open(filename, 'w') as f:
            json.dump(data, f, indent=2)
    except Exception as e:
        log(f"Warning: Failed to save test case history: {e}")

def load_test_case_history(filename: str = "test_case_history.json"):
    """
    Load test case results history from a JSON file

    Args:
        filename: Filename to load from

    Returns:
        dict: Dict mapping test names to list of results, or empty dict if file doesn't exist
    """
    try:
        if os.path.exists(filename):
            with open(filename, 'r') as f:
                data = json.load(f)
                return data.get('test_cases', {})
        return {}
    except Exception as e:
        log(f"Warning: Failed to load test case history: {e}")
        return {}

def check_consecutive_failures(test_name: str, results: list, threshold: int = 3, logfile: str = None):
    """
    Check if a test case has failed consecutively beyond threshold

    Args:
        test_name: Name of the test case
        results: List of recent results (True = Pass, False = Fail)
        threshold: Number of consecutive failures to trigger alert (default: 3)
        logfile: Log file path

    Returns:
        tuple: (is_failing: bool, consecutive_count: int, details: dict)
    """
    if not results:
        return False, 0, {'reason': 'no_data'}

    # Count consecutive failures from the end (most recent)
    consecutive_failures = 0
    for result in reversed(results):
        if not result:  # Failed
            consecutive_failures += 1
        else:
            break

    is_failing = consecutive_failures >= threshold

    details = {
        'test_name': test_name,
        'consecutive_failures': consecutive_failures,
        'threshold': threshold,
        'total_runs': len(results),
        'recent_results': results[-10:]  # Last 10 runs for context
    }

    if is_failing:
        log(f"Consecutive failure alert: '{test_name}' failed {consecutive_failures} times in a row (threshold: {threshold})", logfile)

    return is_failing, consecutive_failures, details

def check_fishers_exact_test(test_name: str, all_results: list,
                              historical_window: int = 20, recent_window: int = 10,
                              alpha: float = 0.05, num_tests: int = 45, logfile: str = None):
    """
    Use Fisher's Exact Test to detect if test is failing statistically more often than before

    Args:
        test_name: Name of the test case
        all_results: Complete list of results (True = Pass, False = Fail)
        historical_window: Number of older runs to use as baseline (default: 20)
        recent_window: Number of recent runs to compare (default: 10)
        alpha: Significance level before Bonferroni correction (default: 0.05)
        num_tests: Total number of test cases for Bonferroni correction (default: 45)
        logfile: Log file path

    Returns:
        tuple: (is_regression: bool, p_value: float, details: dict)
    """
    if not SCIPY_AVAILABLE:
        log(f"Fisher's Exact Test: scipy not available, skipping test '{test_name}'", logfile)
        return False, 1.0, {'reason': 'scipy_not_available'}

    # Need enough data for both windows
    min_required = historical_window + recent_window
    if len(all_results) < min_required:
        return False, 1.0, {
            'reason': 'insufficient_data',
            'required': min_required,
            'available': len(all_results)
        }

    # Split into historical and recent
    # Historical: older runs (not including the most recent runs)
    # Recent: most recent runs
    historical_results = all_results[-(historical_window + recent_window):-recent_window]
    recent_results = all_results[-recent_window:]

    # Build contingency table
    hist_pass = sum(historical_results)
    hist_fail = len(historical_results) - hist_pass
    recent_pass = sum(recent_results)
    recent_fail = len(recent_results) - recent_pass

    # If there are no failures in either period, no regression
    if hist_fail == 0 and recent_fail == 0:
        return False, 1.0, {
            'reason': 'no_failures',
            'historical_pass_rate': 1.0,
            'recent_pass_rate': 1.0
        }

    # Contingency table:
    # [[historical_pass, historical_fail],
    #  [recent_pass, recent_fail]]
    table = [
        [hist_pass, hist_fail],
        [recent_pass, recent_fail]
    ]

    try:
        # Fisher's exact test
        # alternative='greater' tests if recent has MORE failures (worse pass rate)
        oddsratio, p_value = fisher_exact(table, alternative='greater')

        # Bonferroni correction for multiple testing
        adjusted_alpha = alpha / num_tests

        is_regression = p_value < adjusted_alpha

        historical_pass_rate = hist_pass / len(historical_results) if len(historical_results) > 0 else 0
        recent_pass_rate = recent_pass / len(recent_results) if len(recent_results) > 0 else 0

        details = {
            'test_name': test_name,
            'historical_window': historical_window,
            'recent_window': recent_window,
            'historical_pass_rate': historical_pass_rate,
            'recent_pass_rate': recent_pass_rate,
            'historical_passes': hist_pass,
            'historical_fails': hist_fail,
            'recent_passes': recent_pass,
            'recent_fails': recent_fail,
            'odds_ratio': oddsratio,
            'p_value': p_value,
            'alpha': alpha,
            'adjusted_alpha': adjusted_alpha,
            'bonferroni_correction': num_tests
        }

        if is_regression:
            log(f"Fisher's Exact Test: Regression detected for '{test_name}'", logfile)
            log(f"  Historical pass rate: {historical_pass_rate*100:.1f}% ({hist_pass}/{len(historical_results)})", logfile)
            log(f"  Recent pass rate: {recent_pass_rate*100:.1f}% ({recent_pass}/{len(recent_results)})", logfile)
            log(f"  p-value: {p_value:.6f}, adjusted alpha: {adjusted_alpha:.6f}", logfile)
            log(f"  Odds ratio: {oddsratio:.2f}", logfile)

        return is_regression, p_value, details

    except Exception as e:
        log(f"Error in Fisher's Exact Test for '{test_name}': {e}", logfile)
        return False, 1.0, {'error': str(e)}

def bayesian_anomaly_detection(pass_rates: list, current_rate: float,
                               threshold: float = 2.0, logfile: str = None):
    """
    Use Bayesian approach to detect if current pass rate is anomalous

    Uses a simple Bayesian outlier detection based on:
    - Prior: Normal distribution estimated from historical data
    - Likelihood: How likely is the current observation given the prior
    - Detection: Flag as anomaly if observation is beyond threshold standard deviations

    Args:
        pass_rates: Historical pass rates (must have at least 3 data points)
        current_rate: Current pass rate to check
        threshold: Number of standard deviations for anomaly threshold (default: 3.0)
        logfile: Log file path

    Returns:
        tuple: (is_anomaly: bool, probability_score: float, details: dict)
    """
    if len(pass_rates) < 3:
        log("Bayesian detection: Not enough historical data (need at least 3 runs)", logfile)
        return (False, 0.0, {
            'reason': 'insufficient_data',
            'data_points': len(pass_rates)
        })

    try:
        # Calculate prior distribution parameters (mean and std)
        mean = statistics.mean(pass_rates)
        stdev = statistics.stdev(pass_rates)

        # Handle case where stdev is very small (all values very similar)
        if stdev < 0.01:
            stdev = 0.5  # Use small default to avoid division by zero

        # Calculate z-score (number of standard deviations from mean)
        z_score = abs(current_rate - mean) / stdev

        # Calculate probability using normal distribution
        # P(x) = (1 / (σ√(2π))) * e^(-0.5 * ((x-μ)/σ)^2)
        probability = (1 / (stdev * math.sqrt(2 * math.pi))) * \
                     math.exp(-0.5 * ((current_rate - mean) / stdev) ** 2)

        # Flag as anomaly if beyond threshold standard deviations
        is_anomaly = z_score > threshold

        details = {
            'mean': mean,
            'stdev': stdev,
            'z_score': z_score,
            'threshold': threshold,
            'current_rate': current_rate,
            'deviation': abs(current_rate - mean),
            'data_points': len(pass_rates)
        }

        if is_anomaly:
            log(f"Bayesian detection: ANOMALY detected!", logfile)
            log(f"  Current rate: {current_rate:.2f}%", logfile)
            log(f"  Historical mean: {mean:.2f}%", logfile)
            log(f"  Historical stdev: {stdev:.2f}", logfile)
            log(f"  Z-score: {z_score:.2f} (threshold: {threshold})", logfile)
            log(f"  Deviation: {abs(current_rate - mean):.2f}%", logfile)
        else:
            log(f"Bayesian detection: No anomaly (z-score: {z_score:.2f}, threshold: {threshold})", logfile)

        return (is_anomaly, probability, details)

    except Exception as e:
        log(f"Error in Bayesian anomaly detection: {e}", logfile)
        return (False, 0.0, {'error': str(e)})

def check_olmv1_health(logfile: str = None):
    """
    Check if OLM v1 is running properly in the cluster
    Checks: openshift-catalogd, openshift-operator-controller, openshift-cluster-olm-operator namespaces
    and clusteroperator/olm

    Returns:
        tuple: (is_healthy: bool, error_messages: list, detailed_logs: dict)
            - is_healthy: True if all checks pass, False otherwise
            - error_messages: List of human-readable error messages
            - detailed_logs: Dict with keys:
                - 'commands_executed': List of all commands run
                - 'error_outputs': List of dicts with stderr, stdout, returncode for failed commands
                - 'pod_logs': Reserved for future pod log collection
    """
    errors = []
    detailed_logs = {
        'commands_executed': [],
        'error_outputs': [],
        'pod_logs': []
    }

    try:
        # Check ClusterOperator olm
        log("Checking OLM v1 health: ClusterOperator olm...", logfile)
        cmd = ["oc", "get", "clusteroperator", "olm", "-o", "json"]
        detailed_logs['commands_executed'].append(' '.join(cmd))

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode != 0:
            error_msg = f"Failed to get ClusterOperator olm: {result.stderr}"
            errors.append(error_msg)
            detailed_logs['error_outputs'].append({
                'command': ' '.join(cmd),
                'stderr': result.stderr,
                'stdout': result.stdout,
                'returncode': result.returncode
            })
        else:
            try:
                co_data = json.loads(result.stdout)
                conditions = co_data.get('status', {}).get('conditions', [])

                # Check Available condition
                available = None
                progressing = None
                degraded = None

                for condition in conditions:
                    if condition.get('type') == 'Available':
                        available = condition
                    elif condition.get('type') == 'Progressing':
                        progressing = condition
                    elif condition.get('type') == 'Degraded':
                        degraded = condition

                if available and available.get('status') != 'True':
                    errors.append(f"ClusterOperator olm is not Available: {available.get('reason', 'Unknown')} - {available.get('message', '')}")

                if degraded and degraded.get('status') == 'True':
                    errors.append(f"ClusterOperator olm is Degraded: {degraded.get('reason', 'Unknown')} - {degraded.get('message', '')}")

            except json.JSONDecodeError as e:
                errors.append(f"Failed to parse ClusterOperator olm JSON: {e}")

        # Check namespaces exist
        log("Checking OLM v1 health: required namespaces...", logfile)
        required_namespaces = [
            "openshift-catalogd",
            "openshift-operator-controller",
            "openshift-cluster-olm-operator"
        ]

        for ns in required_namespaces:
            cmd = ["oc", "get", "namespace", ns]
            detailed_logs['commands_executed'].append(' '.join(cmd))
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30
            )
            if result.returncode != 0:
                errors.append(f"Required namespace '{ns}' not found")
                detailed_logs['error_outputs'].append({
                    'command': ' '.join(cmd),
                    'stderr': result.stderr,
                    'stdout': result.stdout,
                    'returncode': result.returncode
                })

        # Check pods in openshift-catalogd
        log("Checking OLM v1 health: openshift-catalogd pods...", logfile)
        cmd = ["oc", "get", "pods", "-n", "openshift-catalogd", "-o", "json"]
        detailed_logs['commands_executed'].append(' '.join(cmd))
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode != 0:
            errors.append(f"Failed to get pods in openshift-catalogd: {result.stderr}")
            detailed_logs['error_outputs'].append({
                'command': ' '.join(cmd),
                'stderr': result.stderr,
                'stdout': result.stdout,
                'returncode': result.returncode
            })
        else:
            try:
                pods_data = json.loads(result.stdout)
                if not pods_data.get('items'):
                    errors.append("No pods found in openshift-catalogd namespace")
                else:
                    for pod in pods_data['items']:
                        pod_name = pod['metadata']['name']
                        pod_status = pod['status']['phase']

                        if pod_status not in ['Running', 'Succeeded']:
                            errors.append(f"Pod '{pod_name}' in openshift-catalogd is not Running (status: {pod_status})")
                        elif pod_status == 'Running':
                            # Check container statuses
                            container_statuses = pod['status'].get('containerStatuses', [])
                            for container in container_statuses:
                                if not container.get('ready', False):
                                    errors.append(f"Pod '{pod_name}' in openshift-catalogd container '{container['name']}' is not ready")
            except json.JSONDecodeError as e:
                errors.append(f"Failed to parse openshift-catalogd pods JSON: {e}")

        # Check pods in openshift-operator-controller
        log("Checking OLM v1 health: openshift-operator-controller pods...", logfile)
        cmd = ["oc", "get", "pods", "-n", "openshift-operator-controller", "-o", "json"]
        detailed_logs['commands_executed'].append(' '.join(cmd))
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode != 0:
            errors.append(f"Failed to get pods in openshift-operator-controller: {result.stderr}")
            detailed_logs['error_outputs'].append({
                'command': ' '.join(cmd),
                'stderr': result.stderr,
                'stdout': result.stdout,
                'returncode': result.returncode
            })
        else:
            try:
                pods_data = json.loads(result.stdout)
                if not pods_data.get('items'):
                    errors.append("No pods found in openshift-operator-controller namespace")
                else:
                    for pod in pods_data['items']:
                        pod_name = pod['metadata']['name']
                        pod_status = pod['status']['phase']

                        if pod_status not in ['Running', 'Succeeded']:
                            errors.append(f"Pod '{pod_name}' in openshift-operator-controller is not Running (status: {pod_status})")
                        elif pod_status == 'Running':
                            # Check container statuses
                            container_statuses = pod['status'].get('containerStatuses', [])
                            for container in container_statuses:
                                if not container.get('ready', False):
                                    errors.append(f"Pod '{pod_name}' in openshift-operator-controller container '{container['name']}' is not ready")
            except json.JSONDecodeError as e:
                errors.append(f"Failed to parse openshift-operator-controller pods JSON: {e}")

        # Check pods in openshift-cluster-olm-operator
        log("Checking OLM v1 health: openshift-cluster-olm-operator pods...", logfile)
        cmd = ["oc", "get", "pods", "-n", "openshift-cluster-olm-operator", "-o", "json"]
        detailed_logs['commands_executed'].append(' '.join(cmd))
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode != 0:
            errors.append(f"Failed to get pods in openshift-cluster-olm-operator: {result.stderr}")
            detailed_logs['error_outputs'].append({
                'command': ' '.join(cmd),
                'stderr': result.stderr,
                'stdout': result.stdout,
                'returncode': result.returncode
            })
        else:
            try:
                pods_data = json.loads(result.stdout)
                if not pods_data.get('items'):
                    errors.append("No pods found in openshift-cluster-olm-operator namespace")
                else:
                    for pod in pods_data['items']:
                        pod_name = pod['metadata']['name']
                        pod_status = pod['status']['phase']

                        if pod_status not in ['Running', 'Succeeded']:
                            errors.append(f"Pod '{pod_name}' in openshift-cluster-olm-operator is not Running (status: {pod_status})")
                        elif pod_status == 'Running':
                            # Check container statuses
                            container_statuses = pod['status'].get('containerStatuses', [])
                            for container in container_statuses:
                                if not container.get('ready', False):
                                    errors.append(f"Pod '{pod_name}' in openshift-cluster-olm-operator container '{container['name']}' is not ready")
            except json.JSONDecodeError as e:
                errors.append(f"Failed to parse openshift-cluster-olm-operator pods JSON: {e}")

        # Check CRDs
        log("Checking OLM v1 health: required CRDs...", logfile)
        required_crds = [
            "clustercatalogs.olm.operatorframework.io",
            "clusterextensions.olm.operatorframework.io",
            "olms.operator.openshift.io"
        ]

        for crd in required_crds:
            cmd = ["oc", "get", "crd", crd]
            detailed_logs['commands_executed'].append(' '.join(cmd))
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30
            )
            if result.returncode != 0:
                errors.append(f"Required CRD '{crd}' not found")
                detailed_logs['error_outputs'].append({
                    'command': ' '.join(cmd),
                    'stderr': result.stderr,
                    'stdout': result.stdout,
                    'returncode': result.returncode
                })

        # Check ClusterCatalogs
        log("Checking OLM v1 health: ClusterCatalogs...", logfile)
        cmd = ["oc", "get", "clustercatalogs", "-o", "json"]
        detailed_logs['commands_executed'].append(' '.join(cmd))
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            try:
                catalogs_data = json.loads(result.stdout)
                if catalogs_data.get('items'):
                    # Check if any catalog is not ready
                    for catalog in catalogs_data['items']:
                        catalog_name = catalog['metadata']['name']
                        conditions = catalog.get('status', {}).get('conditions', [])

                        # Look for Serving condition
                        serving_condition = None
                        for condition in conditions:
                            if condition.get('type') == 'Serving':
                                serving_condition = condition
                                break

                        if serving_condition:
                            if serving_condition.get('status') != 'True':
                                errors.append(f"ClusterCatalog '{catalog_name}' is not Serving: {serving_condition.get('reason', 'Unknown')} - {serving_condition.get('message', '')}")
                        else:
                            errors.append(f"ClusterCatalog '{catalog_name}' has no Serving condition")
            except json.JSONDecodeError as e:
                errors.append(f"Failed to parse ClusterCatalogs JSON: {e}")

    except subprocess.TimeoutExpired as e:
        errors.append(f"Timeout while checking OLM v1 health: {e}")
        detailed_logs['error_outputs'].append({
            'error': 'TimeoutExpired',
            'message': str(e)
        })
    except Exception as e:
        errors.append(f"Unexpected error while checking OLM v1 health: {e}")
        detailed_logs['error_outputs'].append({
            'error': type(e).__name__,
            'message': str(e)
        })

    is_healthy = len(errors) == 0

    if is_healthy:
        log("OLM v1 health check: PASSED", logfile)
    else:
        log(f"OLM v1 health check: FAILED with {len(errors)} error(s)", logfile)
        for error in errors:
            log(f"  - {error}", logfile)

    return is_healthy, errors, detailed_logs

def parse_junit_results(results_dir: str = "./results", return_details: bool = False, return_test_cases: bool = False):
    """
    Parse JUnit XML result files and return pass rate
    Only parses the latest JUnit XML file

    Args:
        results_dir: Directory containing JUnit XML files
        return_details: If True, include failed test details
        return_test_cases: If True, return all individual test case results

    Returns:
        Without return_test_cases: (total_tests, passed_tests, pass_rate, failures, errors, skipped) or
                                   (total_tests, passed_tests, pass_rate, failures, errors, skipped, failed_tests) if return_details=True
        With return_test_cases: (..., test_case_results) where test_case_results is dict mapping test names to True/False
    """
    try:
        xml_files = glob.glob(os.path.join(results_dir, "junit_e2e_*.xml"))
        if not xml_files:
            return None

        # Select only the latest XML file
        latest_xml = max(xml_files, key=os.path.getmtime)

        try:
            tree = ET.parse(latest_xml)
            root = tree.getroot()

            # Count actual test cases
            total_tests = 0
            total_failures = 0
            total_errors = 0
            total_skipped = 0
            failed_tests = []
            test_case_results = {}  # Map test name -> True (pass) / False (fail)

            # Iterate through all testcase elements
            testcases = root.findall('.//testcase')

            for testcase in testcases:
                # Skip monitoring tests, they don't count in official statistics
                name = testcase.get('name', '')
                if 'Monitor cluster while tests execute' in name:
                    continue

                total_tests += 1

                # Determine if test passed or failed
                is_passed = True

                # Check for failure element
                if testcase.find('failure') is not None:
                    total_failures += 1
                    is_passed = False
                    if return_details:
                        failed_tests.append({
                            'name': name,
                            'type': 'failure',
                            'message': testcase.find('failure').get('message', 'No message')
                        })
                # Check for error element
                elif testcase.find('error') is not None:
                    total_errors += 1
                    is_passed = False
                    if return_details:
                        failed_tests.append({
                            'name': name,
                            'type': 'error',
                            'message': testcase.find('error').get('message', 'No message')
                        })
                # Check for skipped element
                elif testcase.find('skipped') is not None:
                    total_skipped += 1
                    is_passed = False  # Treat skipped as not passed

                # Store test case result
                if return_test_cases:
                    test_case_results[name] = is_passed

            if total_tests == 0:
                return None

            passed_tests = total_tests - total_failures - total_errors - total_skipped
            pass_rate = (passed_tests / total_tests) * 100

            # Build return value based on parameters
            base_result = (total_tests, passed_tests, pass_rate, total_failures, total_errors, total_skipped)

            if return_test_cases and return_details:
                return (*base_result, failed_tests, test_case_results)
            elif return_test_cases:
                return (*base_result, test_case_results)
            elif return_details:
                return (*base_result, failed_tests)
            else:
                return base_result

        except Exception as e:
            print(f"Warning: Failed to parse {latest_xml}: {e}")
            return None

    except Exception as e:
        print(f"Error parsing JUnit results: {e}")
        return None

def run_kube_burner_ocp(binary: str, extra_args, logfile: str = None):
    cmd = [
        binary,
        "olm",
        "--log-level=debug",
        "--qps=20",
        "--burst=20",
        "--gc=true",
        "--iterations=10",
    ]
    cmd.extend(extra_args)

    log(f"Running: {' '.join(cmd)}", logfile)
    save_command(cmd)

    result = subprocess.run(cmd, text=True, capture_output=True)
    log(f"Process finished with return code {result.returncode}", logfile)
    if result.stdout:
        log(f"STDOUT:\n{result.stdout}", logfile)
    if result.stderr:
        log(f"STDERR:\n{result.stderr}", logfile)

def start_kube_burner_background(binary: str, extra_args, logfile: str = None):
    """
    Start kube-burner-ocp in the background
    Returns: subprocess.Popen object
    """
    cmd = [
        binary,
        "olm",
        "--log-level=debug",
        "--qps=20",
        "--burst=20",
        "--gc=true",
    ]
    cmd.extend(extra_args)

    log(f"Starting kube-burner-ocp in background: {' '.join(cmd)}", logfile)
    save_command(cmd)

    # Open log files for background process
    stdout_file = open("kube-burner.stdout.log", "a")
    stderr_file = open("kube-burner.stderr.log", "a")

    process = subprocess.Popen(
        cmd,
        stdout=stdout_file,
        stderr=stderr_file,
        text=True
    )

    log(f"kube-burner-ocp started in background with PID {process.pid}", logfile)
    log(f"kube-burner-ocp stdout: kube-burner.stdout.log", logfile)
    log(f"kube-burner-ocp stderr: kube-burner.stderr.log", logfile)

    return process, stdout_file, stderr_file

def run_olmv1_tests_ext(binary: str, logfile: str = None, show_stats: bool = False,
                         suite: str = None, include_patterns: list = None, exclude_patterns: list = None,
                         max_concurrency: int = 6):
    """
    Run tests using olmv1-tests-ext binary.

    Args:
        binary: Path to olmv1-tests-ext binary
        logfile: Log file path (optional)
        show_stats: Whether to show statistics
        suite: Test suite name (e.g., 'olmv1/extended/candidate/parallel')
        include_patterns: List of patterns to include in test names
        exclude_patterns: List of patterns to exclude from test names
        max_concurrency: Maximum number of tests to run in parallel (default: 6)

    Returns:
        float: Pass rate if show_stats is True, else None
    """
    try:
        if include_patterns is None:
            include_patterns = []
        # For olmv1-tests-ext, the default "OLM v1" pattern from extended-platform-tests
        # doesn't match because test names use "[sig-olmv1]" format.
        # Since all tests in olmv1-tests-ext are OLM v1 tests, we skip this filter.
        if include_patterns == ["OLM v1"]:
            include_patterns = []
            log("Note: Ignoring default 'OLM v1' include pattern (all olmv1-tests-ext tests are OLM v1)", logfile)
        if exclude_patterns is None:
            exclude_patterns = ["DEPRECATED", "VMonly", "Stress", "Disruptive", "ChkUpgrade"]

        # Ensure results directory exists
        os.makedirs("./results", exist_ok=True)

        # Generate unique junit filename with timestamp
        # Use junit_e2e_* pattern to match parse_junit_results() expectations
        timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
        junit_file = f"./results/junit_e2e_{timestamp}.xml"

        if suite:
            # Use run-suite command with junit output
            run_cmd = [
                binary, "run-suite", suite,
                "-j", junit_file,
                "-c", str(max_concurrency)
            ]
            log(f"Running suite: {' '.join(run_cmd)}", logfile)

            result = subprocess.run(
                run_cmd,
                text=True,
                capture_output=True
            )
        else:
            # List all tests and filter based on include/exclude patterns
            list_cmd = [binary, "list", "tests"]
            log(f"Running: {' '.join(list_cmd)}", logfile)
            list_result = subprocess.run(
                list_cmd, capture_output=True, text=True, check=True
            )

            # Parse JSON output
            tests_json = json.loads(list_result.stdout)
            test_names = []
            for test in tests_json:
                name = test.get('name', '')
                # Check if name contains ALL include patterns (if any)
                if include_patterns and not all(pattern in name for pattern in include_patterns):
                    continue
                # Check if name doesn't contain any exclude patterns
                if any(pattern in name for pattern in exclude_patterns):
                    continue
                test_names.append(name)

            if not test_names:
                log("No matching tests found.", logfile)
                return None

            log(f"Found {len(test_names)} matching tests.", logfile)

            # Build run-test command with all test names
            run_cmd = [
                binary, "run-test",
                "-c", str(max_concurrency),
                "-o", "json"
            ]
            for name in test_names:
                run_cmd.extend(["-n", name])

            log(f"Running: {binary} run-test -c {max_concurrency} -o json -n <{len(test_names)} tests>", logfile)

            result = subprocess.run(
                run_cmd,
                text=True,
                capture_output=True
            )

            # Parse JSON output and generate junit XML
            if result.stdout:
                try:
                    # Save raw JSON output for debugging
                    json_debug_file = junit_file.replace('.xml', '.json')
                    with open(json_debug_file, 'w') as f:
                        f.write(result.stdout)
                    log(f"Saved raw JSON output to: {json_debug_file}", logfile)

                    test_results = json.loads(result.stdout)
                    _generate_junit_from_olmv1_results(test_results, junit_file, logfile)
                except json.JSONDecodeError as e:
                    log(f"Warning: Could not parse JSON output: {e}", logfile)
                    log(f"Raw output: {result.stdout[:500]}...", logfile)

        log(f"Process finished with return code {result.returncode}", logfile)
        if result.stdout and len(result.stdout) < 2000:
            log(f"STDOUT:\n{result.stdout}", logfile)
        if result.stderr:
            log(f"STDERR:\n{result.stderr}", logfile)

        # Parse and display statistics
        if show_stats:
            stats = parse_junit_results()
            if stats:
                total, passed, pass_rate, failures, errors, skipped = stats
                log("=" * 60, logfile)
                log(f"Test Results Summary:", logfile)
                log(f"  Total Tests:    {total}", logfile)
                log(f"  Passed:         {passed}", logfile)
                log(f"  Failed:         {failures}", logfile)
                log(f"  Errors:         {errors}", logfile)
                log(f"  Skipped:        {skipped}", logfile)
                log(f"  Pass Rate:      {pass_rate:.2f}%", logfile)
                log("=" * 60, logfile)
                return pass_rate
            else:
                log("Warning: Could not parse test results", logfile)
                return None

        return None

    except subprocess.CalledProcessError as e:
        log(f"Error: exited with code {e.returncode}", logfile)
        if e.stdout:
            log(f"STDOUT:\n{e.stdout}", logfile)
        if e.stderr:
            log(f"STDERR:\n{e.stderr}", logfile)
        return None
    except Exception as e:
        log(f"Unexpected error: {e}", logfile)
        import traceback
        log(f"Traceback: {traceback.format_exc()}", logfile)
        return None


def _generate_junit_from_olmv1_results(test_results: list, junit_file: str, logfile: str = None):
    """
    Generate JUnit XML file from olmv1-tests-ext JSON output.

    Args:
        test_results: List of test result dictionaries from olmv1-tests-ext
        junit_file: Path to write junit XML file
        logfile: Log file path (optional)
    """
    try:
        # Create root element
        testsuite = ET.Element('testsuite')

        total_tests = 0
        failures = 0
        errors = 0
        skipped = 0
        total_time = 0.0

        for result in test_results:
            name = result.get('name', 'unknown')
            # olmv1-tests-ext uses 'result' field, not 'status'
            status = result.get('result', result.get('status', 'unknown'))
            duration = result.get('duration', 0)
            # olmv1-tests-ext uses 'output' field for error messages
            message = result.get('output', result.get('message', ''))

            total_tests += 1
            total_time += duration

            testcase = ET.SubElement(testsuite, 'testcase')
            testcase.set('name', name)
            testcase.set('time', str(duration))

            if status == 'passed':
                pass  # No child element needed
            elif status == 'failed':
                failures += 1
                failure = ET.SubElement(testcase, 'failure')
                failure.set('message', message[:500] if message else 'Test failed')
                failure.text = message
            elif status == 'error':
                errors += 1
                error = ET.SubElement(testcase, 'error')
                error.set('message', message[:500] if message else 'Test error')
                error.text = message
            elif status == 'skipped':
                skipped += 1
                skip = ET.SubElement(testcase, 'skipped')
                if message:
                    skip.set('message', message[:500])

        # Set testsuite attributes
        testsuite.set('tests', str(total_tests))
        testsuite.set('failures', str(failures))
        testsuite.set('errors', str(errors))
        testsuite.set('skipped', str(skipped))
        testsuite.set('time', str(total_time))
        testsuite.set('name', 'olmv1-tests-ext')

        # Write to file
        tree = ET.ElementTree(testsuite)
        tree.write(junit_file, encoding='unicode', xml_declaration=True)
        log(f"Generated JUnit XML: {junit_file}", logfile)

    except Exception as e:
        log(f"Error generating JUnit XML: {e}", logfile)


def run_extended_platform_tests(binary: str, logfile: str = None, show_stats: bool = False,
                                  include_patterns: list = None, exclude_patterns: list = None):
    try:
        if include_patterns is None:
            include_patterns = ["OLM v1"]
        if exclude_patterns is None:
            exclude_patterns = ["DEPRECATED", "VMonly", "Stress", "Disruptive", "ChkUpgrade"]

        dry_run_cmd = [binary, "run", "all", "--dry-run"]
        log(f"Running: {' '.join(dry_run_cmd)}", logfile)
        dry_run = subprocess.run(
            dry_run_cmd, capture_output=True, text=True, check=True
        )

        # Filter tests based on include and exclude patterns
        tests = []
        for line in dry_run.stdout.splitlines():
            # Check if line contains ALL include patterns
            if all(include_pattern in line for include_pattern in include_patterns):
                # Check if line doesn't contain any exclude patterns
                if not any(exclude_pattern in line for exclude_pattern in exclude_patterns):
                    tests.append(line)
        if not tests:
            log("No matching tests found.", logfile)
            return None

        log(f"Found {len(tests)} matching tests.", logfile)

        run_cmd = [
            binary, "run",
            "--junit-dir=./results",
            "--max-parallel-tests", "6",
            "-f", "-"
        ]
        log(f"Running: {' '.join(run_cmd)}", logfile)

        result = subprocess.run(
            run_cmd,
            input="\n".join(tests),
            text=True,
            capture_output=True
        )

        log(f"Process finished with return code {result.returncode}", logfile)
        if result.stdout:
            log(f"STDOUT:\n{result.stdout}", logfile)
        if result.stderr:
            log(f"STDERR:\n{result.stderr}", logfile)

        # Parse and display statistics
        if show_stats:
            stats = parse_junit_results()
            if stats:
                total, passed, pass_rate, failures, errors, skipped = stats
                log("=" * 60, logfile)
                log(f"Test Results Summary:", logfile)
                log(f"  Total Tests:    {total}", logfile)
                log(f"  Passed:         {passed}", logfile)
                log(f"  Failed:         {failures}", logfile)
                log(f"  Errors:         {errors}", logfile)
                log(f"  Skipped:        {skipped}", logfile)
                log(f"  Pass Rate:      {pass_rate:.2f}%", logfile)
                log("=" * 60, logfile)
                return pass_rate
            else:
                log("Warning: Could not parse test results", logfile)
                return None

        return None

    except subprocess.CalledProcessError as e:
        log(f"Error: exited with code {e.returncode}", logfile)
        if e.stdout:
            log(f"STDOUT:\n{e.stdout}", logfile)
        if e.stderr:
            log(f"STDERR:\n{e.stderr}", logfile)
        return None
    except Exception as e:
        log(f"Unexpected error: {e}", logfile)
        return None

def main():
    parser = argparse.ArgumentParser(description="Trigger extended-platform-tests, olmv1-tests-ext, or kube-burner-ocp periodically")
    parser.add_argument(
        "-b", "--binary",
        type=str,
        required=True,
        help="Path to the binary (e.g., /path/to/extended-platform-tests, /path/to/olmv1-tests-ext, or /usr/local/bin/kube-burner-ocp)"
    )
    parser.add_argument(
        "-i", "--interval",
        type=int,
        default=600,
        help="Interval between executions in seconds (default: 600 = 10 minutes)"
    )
    parser.add_argument(
        "-l", "--logfile",
        type=str,
        default=None,
        help="Path to log file (optional)"
    )
    parser.add_argument(
        "-o", "--once",
        action="store_true",
        help="Run only once and exit"
    )
    parser.add_argument(
        "-s", "--stats",
        action="store_true",
        help="Show test pass rate statistics and calculate average across runs"
    )
    parser.add_argument(
        "-d", "--days",
        type=float,
        default=None,
        help="Total duration to run in days (e.g., 5 for 5 days, 0.5 for 12 hours)"
    )
    parser.add_argument(
        "-m", "--max-runs",
        type=int,
        default=None,
        help="Maximum number of runs before exiting (default: unlimited)"
    )
    parser.add_argument(
        "--slack-webhook",
        type=str,
        default=None,
        help="Slack Webhook URL for notifications (required for Slack alerts)"
    )
    parser.add_argument(
        "--min-pass-rate",
        type=float,
        default=85.0,
        help="Minimum pass rate threshold in percentage (default: 85.0). Only checked on first run"
    )
    parser.add_argument(
        "--anomaly-threshold",
        type=float,
        default=3.0,
        help="Bayesian anomaly detection threshold in standard deviations (default: 3.0). Higher values = less sensitive"
    )
    parser.add_argument(
        "--include-patterns",
        type=str,
        nargs="+",
        default=["OLM v1"],
        help="Test case patterns to include (default: 'OLM v1'). Only tests containing ALL of these strings will be selected. Example: --include-patterns high olm jiazha"
    )
    parser.add_argument(
        "--exclude-patterns",
        type=str,
        nargs="+",
        default=["DEPRECATED", "VMonly", "Stress", "Disruptive", "ChkUpgrade"],
        help="Test case patterns to exclude (default: DEPRECATED VMonly Stress Disruptive ChkUpgrade). Tests containing any of these strings will be excluded"
    )
    parser.add_argument(
        "--suite",
        type=str,
        default=None,
        help="Test suite name for olmv1-tests-ext binary (e.g., 'olmv1/extended/candidate/parallel'). If not specified, tests are filtered by include/exclude patterns"
    )
    parser.add_argument(
        "--max-concurrency",
        type=int,
        default=6,
        help="Maximum number of tests to run in parallel (default: 6)"
    )
    parser.add_argument(
        "--kube-burner-binary",
        type=str,
        default=None,
        help="Path to kube-burner-ocp binary. If specified, kube-burner-ocp will run in the background while executing the main binary"
    )
    parser.add_argument(
        "--kube-burner-args",
        type=str,
        nargs=argparse.REMAINDER,
        default=[],
        help="Extra arguments passed to kube-burner-ocp (only used with --kube-burner-binary)"
    )
    parser.add_argument(
        "--check-olmv1-health",
        action="store_true",
        help="Check OLM v1 health before each test run. Stop and send Slack notification if OLM v1 is not healthy"
    )
    parser.add_argument(
        "--google-sheet-id",
        type=str,
        default=None,
        help="Google Spreadsheet ID (from URL) to write test results"
    )
    parser.add_argument(
        "--google-sheet-name",
        type=str,
        default="OLM Overall Performance Test Report",
        help="Name of the sheet/tab in Google Spreadsheet (default: 'OLM Overall Performance Test Report')"
    )
    parser.add_argument(
        "--google-sheet-start-cell",
        type=str,
        default=None,
        help="Start cell for writing data (e.g., 'B30'). If not specified, appends to the end of the sheet."
    )
    parser.add_argument(
        "--google-sheet-per-run",
        action="store_true",
        help="Write per-run results to Google Sheets after each test run"
    )
    parser.add_argument(
        "--google-sheet-per-run-name",
        type=str,
        default="Long-duration Test Results",
        help="Name of the sheet/tab for per-run raw results (default: 'Long-duration Test Results')"
    )
    parser.add_argument(
        "--google-credentials",
        type=str,
        default=None,
        help="Path to Google service account credentials JSON file (required if --google-sheet-id is specified)"
    )
    parser.add_argument(
        "--disable-fishers-test",
        action="store_true",
        help="Disable Fisher's Exact Test for individual test case regression detection (enabled by default)"
    )
    parser.add_argument(
        "--fishers-historical-window",
        type=int,
        default=20,
        help="Fisher's Exact Test: number of historical runs for baseline (default: 20)"
    )
    parser.add_argument(
        "--fishers-recent-window",
        type=int,
        default=10,
        help="Fisher's Exact Test: number of recent runs to compare (default: 10)"
    )
    parser.add_argument(
        "--fishers-alpha",
        type=float,
        default=0.05,
        help="Fisher's Exact Test: significance level before Bonferroni correction (default: 0.05)"
    )
    parser.add_argument(
        "--enable-consecutive-failure-check",
        action="store_true",
        help="Enable consecutive failure detection for individual test cases (disabled by default)"
    )
    parser.add_argument(
        "--consecutive-failure-threshold",
        type=int,
        default=3,
        help="Alert if a test case fails N consecutive times (default: 3). Only used with --enable-consecutive-failure-check"
    )
    parser.add_argument(
        "--must-gather-on-error",
        action="store_true",
        help="Run 'oc adm must-gather' when the program exits abnormally (errors, health check failures, etc.)"
    )
    parser.add_argument(
        "--must-gather-dir",
        type=str,
        default="./must-gather",
        help="Directory to store must-gather output (default: ./must-gather)"
    )
    args = parser.parse_args()

    binary = args.binary
    binary_name = os.path.basename(binary)

    if not os.path.isfile(binary):
        print(f"Error: {binary} does not exist.")
        sys.exit(1)
    if not os.access(binary, os.X_OK):
        print(f"Error: {binary} is not executable.")
        sys.exit(1)

    # Validate kube-burner binary if specified
    if args.kube_burner_binary:
        if not os.path.isfile(args.kube_burner_binary):
            print(f"Error: kube-burner binary {args.kube_burner_binary} does not exist.")
            sys.exit(1)
        if not os.access(args.kube_burner_binary, os.X_OK):
            print(f"Error: kube-burner binary {args.kube_burner_binary} is not executable.")
            sys.exit(1)

    # Validate Google Sheets parameters
    if args.google_sheet_id:
        if not args.google_credentials:
            print("Error: --google-credentials is required when --google-sheet-id is specified.")
            sys.exit(1)
        if not os.path.isfile(args.google_credentials):
            print(f"Error: Google credentials file not found: {args.google_credentials}")
            sys.exit(1)
        if not GOOGLE_SHEETS_AVAILABLE:
            print("Error: Google Sheets API libraries not installed.")
            print("Install with: pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client")
            sys.exit(1)

    # Track pass rates across multiple runs
    pass_rates = []
    run_count = 0

    # Record start time (for --days parameter)
    start_time = datetime.now()

    # Get OCP version if writing to Google Sheets
    ocp_version = "Unknown"
    if args.google_sheet_id:
        ocp_version = get_ocp_version(args.logfile)
        log(f"Detected OCP version: {ocp_version}", args.logfile)

    # Calculate end time (if --days is set)
    end_time = None
    if args.days is not None:
        end_time = start_time + timedelta(days=args.days)
        log(f"Will run for {args.days} days (until {end_time.strftime('%Y-%m-%d %H:%M:%S')})", args.logfile)

    # Record maximum run count
    if args.max_runs is not None:
        log(f"Will run maximum {args.max_runs} times", args.logfile)

    # Start kube-burner in background if specified
    kube_burner_process = None
    kube_burner_stdout = None
    kube_burner_stderr = None
    if args.kube_burner_binary:
        kube_burner_process, kube_burner_stdout, kube_burner_stderr = start_kube_burner_background(
            args.kube_burner_binary, args.kube_burner_args, args.logfile
        )

    # Track if exit was abnormal (for must-gather)
    abnormal_exit = False
    exit_reason = None

    try:
        while True:
            run_count += 1

            # Check OLM v1 health if flag is set
            if args.check_olmv1_health:
                log("=" * 60, args.logfile)
                log(f"Performing OLM v1 health check (run #{run_count})...", args.logfile)
                is_healthy, health_errors, detailed_logs = check_olmv1_health(args.logfile)

                if not is_healthy:
                    log("!" * 60, args.logfile)
                    log("CRITICAL: OLM v1 is not healthy! Stopping execution.", args.logfile)
                    log("!" * 60, args.logfile)

                    # Send Slack notification
                    if args.slack_webhook:
                        error_list = "\n".join([f"• {error}" for error in health_errors[:15]])
                        if len(health_errors) > 15:
                            error_list += f"\n... and {len(health_errors) - 15} more errors"

                        # Build detailed error logs section
                        detailed_error_section = ""
                        if detailed_logs['error_outputs']:
                            detailed_error_section = "\n\n*Detailed Error Logs:*\n"
                            # Show first 5 error outputs to avoid Slack message size limits
                            for i, error_output in enumerate(detailed_logs['error_outputs'][:5], 1):
                                if 'command' in error_output:
                                    detailed_error_section += f"\n{i}. Command: `{error_output['command']}`\n"
                                    if error_output.get('stderr'):
                                        # Truncate stderr to first 200 chars
                                        stderr_preview = error_output['stderr'][:200]
                                        if len(error_output['stderr']) > 200:
                                            stderr_preview += "..."
                                        detailed_error_section += f"   Error: {stderr_preview}\n"
                                    detailed_error_section += f"   Return Code: {error_output.get('returncode', 'N/A')}\n"
                                elif 'error' in error_output:
                                    # Exception error
                                    detailed_error_section += f"\n{i}. Exception: {error_output['error']}\n"
                                    detailed_error_section += f"   Message: {error_output['message']}\n"

                            if len(detailed_logs['error_outputs']) > 5:
                                detailed_error_section += f"\n... and {len(detailed_logs['error_outputs']) - 5} more error outputs\n"

                        # Build commands executed section
                        commands_section = ""
                        if detailed_logs['commands_executed']:
                            commands_section = f"\n\n*Commands Executed:* {len(detailed_logs['commands_executed'])} total"

                        slack_message = (
                            f":x: *OLM v1 Health Check Failed - Stopping Tests*\n\n"
                            f"*Run:* #{run_count}\n"
                            f"*Total Errors:* {len(health_errors)}\n\n"
                            f"*Error Summary:*\n{error_list}"
                            f"{detailed_error_section}"
                            f"{commands_section}\n\n"
                            f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                        )
                        send_slack_message(args.slack_webhook, slack_message, args.logfile)

                    log("Exiting due to OLM v1 health check failure.", args.logfile)
                    abnormal_exit = True
                    exit_reason = "OLM v1 health check failed"
                    break

                log("OLM v1 health check passed, proceeding with tests...", args.logfile)
                log("=" * 60, args.logfile)

            if binary_name == "kube-burner-ocp":
                run_kube_burner_ocp(binary, args.kube_burner_args, args.logfile)
            elif binary_name == "olmv1-tests-ext":
                pass_rate = run_olmv1_tests_ext(binary, args.logfile, args.stats,
                                                args.suite, args.include_patterns,
                                                args.exclude_patterns, args.max_concurrency)
            else:
                pass_rate = run_extended_platform_tests(binary, args.logfile, args.stats,
                                                       args.include_patterns, args.exclude_patterns)

            # Common logic for all test binaries (extended-platform-tests and olmv1-tests-ext)
            if binary_name != "kube-burner-ocp":
                if args.stats and pass_rate is not None:
                    # Add current pass rate to history
                    pass_rates.append(pass_rate)

                    # Save pass rates to file after each run
                    save_pass_rates(pass_rates)

                    # Write failure details to Google Sheets if configured
                    if args.google_sheet_id and args.google_credentials:
                        detailed_stats = parse_junit_results(return_details=True)
                        if detailed_stats and len(detailed_stats) == 7:
                            total, passed, pr, failures, errors, skipped, failed_tests = detailed_stats
                            if failed_tests:  # Only write if there are failures
                                timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                                write_failure_details_to_google_sheet(
                                    args.google_sheet_id,
                                    "Failed Tests",  # Separate sheet for failed tests
                                    args.google_credentials,
                                    run_count,
                                    timestamp,
                                    failed_tests,
                                    args.logfile
                                )

                    # Bayesian anomaly detection (skip first 3 runs to build baseline)
                    # Do this BEFORE writing to Google Sheets so we can mark anomalies
                    current_run_is_anomaly = False
                    if run_count > 3:
                        # Use all previous runs except the current one for baseline
                        historical_rates = pass_rates[:-1]
                        is_anomaly, probability, details = bayesian_anomaly_detection(
                            historical_rates,
                            pass_rate,
                            args.anomaly_threshold,
                            args.logfile
                        )

                        if is_anomaly:
                            current_run_is_anomaly = True
                            log("!" * 60, args.logfile)
                            log(f"ANOMALY DETECTED in run #{run_count}!", args.logfile)
                            log("!" * 60, args.logfile)

                            # Send Slack notification
                            if args.slack_webhook:
                                slack_message = (
                                    f":warning: *Bayesian Anomaly Detection Alert*\n\n"
                                    f"*Run:* #{run_count}\n"
                                    f"*Current Pass Rate:* {pass_rate:.2f}%\n"
                                    f"*Historical Mean:* {details['mean']:.2f}%\n"
                                    f"*Historical Std Dev:* {details['stdev']:.2f}\n"
                                    f"*Z-Score:* {details['z_score']:.2f} (threshold: {details['threshold']})\n"
                                    f"*Deviation:* {details['deviation']:.2f}%\n"
                                    f"*Data Points:* {details['data_points']}\n\n"
                                    f"This pass rate is statistically unusual compared to historical data.\n\n"
                                    f"*Recent Pass Rates:* {', '.join([f'{r:.2f}%' for r in pass_rates[-5:]])}\n\n"
                                    f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                                )
                                send_slack_message(args.slack_webhook, slack_message, args.logfile)

                    # Write per-run results to Google Sheets if configured
                    if args.google_sheet_id and args.google_credentials and args.google_sheet_per_run:
                        # Use return_details=True to get failed test information
                        detailed_stats_for_sheet = parse_junit_results(return_details=True)
                        if detailed_stats_for_sheet and len(detailed_stats_for_sheet) == 7:
                            total_tests, passed_tests, pr, failures, errors, skipped, failed_tests_list = detailed_stats_for_sheet
                            current_date = datetime.now().strftime('%Y-%m-%d')
                            started_time_str = start_time.strftime('%Y-%m-%d %H:%M:%S')
                            write_per_run_results_to_google_sheet(
                                args.google_sheet_id,
                                args.google_sheet_per_run_name,  # Per-run results sheet (customizable)
                                args.google_credentials,
                                run_count,
                                current_date,
                                ocp_version,
                                pass_rate,
                                total_tests,
                                started_time_str,
                                args.logfile,
                                skipped=skipped,
                                failures=failures,
                                errors=errors,
                                failed_tests=failed_tests_list,
                                is_anomaly=current_run_is_anomaly
                            )

                    # Check if first run and pass rate is below minimum threshold
                    if run_count == 1 and pass_rate < args.min_pass_rate:
                        log("!" * 60, args.logfile)
                        log(f"CRITICAL: First run pass rate ({pass_rate:.2f}%) is below minimum threshold ({args.min_pass_rate:.2f}%)!", args.logfile)
                        log("!" * 60, args.logfile)

                        # Get detailed results for Slack notification
                        detailed_stats = parse_junit_results(return_details=True)
                        if detailed_stats and args.slack_webhook:
                            total, passed, pr, failures, errors, skipped, failed_tests = detailed_stats

                            # Build Slack message with test results
                            slack_message = (
                                f":x: *First Run Pass Rate Below Threshold - Stopping Execution*\n\n"
                                f"*Test Results Summary:*\n"
                                f"• Total Tests: {total}\n"
                                f"• Passed: {passed}\n"
                                f"• Failed: {failures}\n"
                                f"• Errors: {errors}\n"
                                f"• Skipped: {skipped}\n"
                                f"• Pass Rate: {pass_rate:.2f}%\n"
                                f"• Minimum Required: {args.min_pass_rate:.2f}%\n\n"
                            )

                            # Add failed test details (limit to first 20 to avoid message being too long)
                            if failed_tests:
                                slack_message += f"*Failed/Error Tests (showing {min(len(failed_tests), 20)} of {len(failed_tests)}):*\n"
                                for i, test in enumerate(failed_tests[:20], 1):
                                    test_name = test['name']
                                    slack_message += f"{i}. [{test['type'].upper()}] {test_name}\n"

                                if len(failed_tests) > 20:
                                    slack_message += f"\n... and {len(failed_tests) - 20} more failures\n"

                            slack_message += f"\n*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"

                            send_slack_message(args.slack_webhook, slack_message, args.logfile)

                        log(f"Exiting due to first run pass rate below minimum threshold.", args.logfile)
                        abnormal_exit = True
                        exit_reason = f"First run pass rate ({pass_rate:.2f}%) below minimum threshold ({args.min_pass_rate:.2f}%)"
                        break

                    # Test case health check (Layer 2 & 3: Fisher's Exact Test + Consecutive Failures)
                    # Fisher's Test is enabled by default (unless --disable-fishers-test)
                    # Consecutive failure check requires --enable-consecutive-failure-check
                    if not args.disable_fishers_test or args.enable_consecutive_failure_check:
                        log("=" * 60, args.logfile)
                        log(f"Performing individual test case health checks (run #{run_count})...", args.logfile)

                        if args.disable_fishers_test:
                            log("Fisher's Exact Test: DISABLED", args.logfile)
                        else:
                            log("Fisher's Exact Test: ENABLED (default)", args.logfile)

                        if args.enable_consecutive_failure_check:
                            log("Consecutive Failure Check: ENABLED", args.logfile)
                        else:
                            log("Consecutive Failure Check: DISABLED (default)", args.logfile)

                        # Get current test case results
                        current_results_with_test_cases = parse_junit_results(return_test_cases=True)
                        if current_results_with_test_cases and len(current_results_with_test_cases) == 7:
                            current_test_case_results = current_results_with_test_cases[6]  # Last element

                            # Load historical test case results
                            test_case_history = load_test_case_history()

                            # Update history with current results
                            for test_name, is_passed in current_test_case_results.items():
                                if test_name not in test_case_history:
                                    test_case_history[test_name] = []
                                test_case_history[test_name].append(is_passed)

                            # Save updated history
                            save_test_case_history(test_case_history)

                            # Count total tests for Bonferroni correction
                            num_tests = len(current_test_case_results)

                            # Check each test case
                            consecutive_failures_detected = []
                            fishers_regressions_detected = []

                            for test_name, results in test_case_history.items():
                                # Layer 3: Consecutive failure detection (optional, needs --enable-consecutive-failure-check)
                                if args.enable_consecutive_failure_check:
                                    is_failing, consecutive_count, consec_details = check_consecutive_failures(
                                        test_name, results, args.consecutive_failure_threshold, args.logfile
                                    )
                                    if is_failing:
                                        consecutive_failures_detected.append((test_name, consecutive_count, consec_details))

                                # Layer 2: Fisher's Exact Test (default, unless --disable-fishers-test)
                                if not args.disable_fishers_test:
                                    is_regression, p_value, fisher_details = check_fishers_exact_test(
                                        test_name, results,
                                        args.fishers_historical_window,
                                        args.fishers_recent_window,
                                        args.fishers_alpha,
                                        num_tests,
                                        args.logfile
                                    )
                                    if is_regression:
                                        fishers_regressions_detected.append((test_name, p_value, fisher_details))

                            # Send Slack notifications for detected issues
                            if args.slack_webhook:
                                # Consecutive failures alert
                                if consecutive_failures_detected:
                                    log(f"Detected {len(consecutive_failures_detected)} test(s) with consecutive failures", args.logfile)
                                    for test_name, count, details in consecutive_failures_detected[:5]:  # Limit to 5 to avoid spam
                                        recent_results_str = ''.join(['✅' if r else '❌' for r in details['recent_results']])
                                        slack_message = (
                                            f":x: *Consecutive Test Failure Alert*\n\n"
                                            f"*Test Case:* {test_name}\n\n"
                                            f"*Consecutive Failures:* {count} (threshold: {details['threshold']})\n"
                                            f"*Total Runs:* {details['total_runs']}\n"
                                            f"*Recent Results:* {recent_results_str}\n\n"
                                            f"*Severity:* HIGH - This test has failed {count} times in a row\n\n"
                                            f"*Run:* #{run_count}\n"
                                            f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                                        )
                                        send_slack_message(args.slack_webhook, slack_message, args.logfile)

                                # Fisher's test regression alert
                                if fishers_regressions_detected:
                                    log(f"Detected {len(fishers_regressions_detected)} test(s) with statistical regression", args.logfile)
                                    for test_name, p_value, details in fishers_regressions_detected[:5]:  # Limit to 5
                                        if 'error' not in details and 'reason' not in details:
                                            slack_message = (
                                                f":warning: *Test Regression Detected (Fisher's Exact Test)*\n\n"
                                                f"*Test Case:* {test_name}\n\n"
                                                f"*Statistical Analysis:*\n"
                                                f"• Historical Pass Rate: {details['historical_pass_rate']*100:.1f}% "
                                                f"({details['historical_passes']}/{details['historical_passes']+details['historical_fails']} runs)\n"
                                                f"• Recent Pass Rate: {details['recent_pass_rate']*100:.1f}% "
                                                f"({details['recent_passes']}/{details['recent_passes']+details['recent_fails']} runs)\n"
                                                f"• Fisher's p-value: {details['p_value']:.6f}\n"
                                                f"• Significance Level: {details['adjusted_alpha']:.6f} (Bonferroni corrected)\n"
                                                f"• Odds Ratio: {details['odds_ratio']:.2f}x\n\n"
                                                f"*Result:* **Statistically Significant Regression**\n\n"
                                                f"*Severity:* MEDIUM - Test is failing more often than before\n\n"
                                                f"*Run:* #{run_count}\n"
                                                f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                                            )
                                            send_slack_message(args.slack_webhook, slack_message, args.logfile)

                            log("Test case health check completed", args.logfile)
                            log("=" * 60, args.logfile)

            # If stats enabled and multiple runs, show average pass rate
            if args.stats and len(pass_rates) > 0:
                avg_pass_rate = sum(pass_rates) / len(pass_rates)
                log("=" * 60, args.logfile)
                log(f"Overall Statistics (after {len(pass_rates)} run(s)):", args.logfile)
                log(f"  Average Pass Rate: {avg_pass_rate:.2f}%", args.logfile)
                if len(pass_rates) > 1:
                    log(f"  Pass Rates: {', '.join([f'{rate:.2f}%' for rate in pass_rates])}", args.logfile)
                log("=" * 60, args.logfile)

            if args.once:
                log("Completed single execution (--once set). Exiting.", args.logfile)
                break

            # Check if maximum run count reached
            if args.max_runs is not None and run_count >= args.max_runs:
                log(f"Reached maximum run count ({args.max_runs}). Exiting.", args.logfile)
                break

            # Check if exceeded specified run days
            if end_time is not None and datetime.now() >= end_time:
                elapsed = datetime.now() - start_time
                log(f"Reached time limit ({args.days} days, elapsed: {elapsed}). Exiting.", args.logfile)
                break

            # Display remaining time/count info
            remaining_info = []
            if args.max_runs is not None:
                remaining_runs = args.max_runs - run_count
                remaining_info.append(f"{remaining_runs} runs remaining")
            if end_time is not None:
                remaining_time = end_time - datetime.now()
                days = remaining_time.days
                hours = remaining_time.seconds // 3600
                minutes = (remaining_time.seconds % 3600) // 60
                remaining_info.append(f"~{days}d {hours}h {minutes}m remaining")

            if remaining_info:
                log(f"Status: {', '.join(remaining_info)}", args.logfile)

            log(f"Waiting {args.interval} seconds before next trigger...", args.logfile)
            time.sleep(args.interval)
    except KeyboardInterrupt:
        log("Received Ctrl+C, exiting gracefully...", args.logfile)
    except Exception as e:
        # Unexpected exception - mark as abnormal exit
        log("!" * 60, args.logfile)
        log(f"CRITICAL: Unexpected exception occurred: {e}", args.logfile)
        log("!" * 60, args.logfile)
        import traceback
        log(f"Traceback: {traceback.format_exc()}", args.logfile)
        abnormal_exit = True
        exit_reason = f"Unexpected exception: {type(e).__name__}: {e}"
    finally:
        # Execute must-gather if abnormal exit and flag is set
        if abnormal_exit and args.must_gather_on_error:
            log("=" * 60, args.logfile)
            log("ABNORMAL EXIT DETECTED - Running must-gather", args.logfile)
            log(f"Exit reason: {exit_reason}", args.logfile)
            log("=" * 60, args.logfile)
            run_must_gather(args.must_gather_dir, args.logfile)

        # Stop background kube-burner process if running
        if kube_burner_process:
            log(f"Stopping background kube-burner-ocp process (PID {kube_burner_process.pid})...", args.logfile)
            kube_burner_process.terminate()
            try:
                kube_burner_process.wait(timeout=10)
                log("kube-burner-ocp process terminated gracefully", args.logfile)
            except subprocess.TimeoutExpired:
                log("kube-burner-ocp process did not terminate, killing it...", args.logfile)
                kube_burner_process.kill()
                kube_burner_process.wait()
                log("kube-burner-ocp process killed", args.logfile)

            # Close log files
            if kube_burner_stdout:
                kube_burner_stdout.close()
            if kube_burner_stderr:
                kube_burner_stderr.close()

        # Display final statistics
        if args.stats and len(pass_rates) > 0:
            avg_pass_rate = sum(pass_rates) / len(pass_rates)
            min_pass_rate_value = min(pass_rates)
            max_pass_rate_value = max(pass_rates)

            log("=" * 60, args.logfile)
            log(f"Final Statistics Summary:", args.logfile)
            log(f"  Total Runs:        {len(pass_rates)}", args.logfile)
            log(f"  Average Pass Rate: {avg_pass_rate:.2f}%", args.logfile)
            log(f"  Min Pass Rate:     {min_pass_rate_value:.2f}%", args.logfile)
            log(f"  Max Pass Rate:     {max_pass_rate_value:.2f}%", args.logfile)
            log(f"  All Pass Rates:    {', '.join([f'{rate:.2f}%' for rate in pass_rates])}", args.logfile)
            log("=" * 60, args.logfile)

            # Write to Google Sheets if configured
            if args.google_sheet_id and args.google_credentials:
                end_time_final = datetime.now()
                test_duration = end_time_final - start_time

                # Format duration as "X days Y hours"
                days = test_duration.days
                hours = test_duration.seconds // 3600
                duration_str = f"{days}d {hours}h"

                # Get total tests and failure details from last run
                last_stats = parse_junit_results(return_details=True)
                total_tests = 0
                total_failures = 0
                total_errors = 0
                total_skipped = 0
                failed_tests_summary = ""

                if last_stats and len(last_stats) == 7:
                    total_tests, passed, pr, failures, errors, skipped, failed_tests_list = last_stats
                    total_failures = failures + errors
                    total_skipped = skipped

                    # Create summary of failed tests (unique test names across runs)
                    if failed_tests_list:
                        failed_names = [t.get('name', 'Unknown') for t in failed_tests_list]
                        failed_tests_summary = "; ".join(failed_names)
                        # Limit to 2000 characters
                        if len(failed_tests_summary) > 2000:
                            failed_tests_summary = failed_tests_summary[:1997] + "..."

                # Determine PASS/FAIL status based on avg_pass_rate vs min_pass_rate threshold
                status = "PASS" if avg_pass_rate >= args.min_pass_rate else "FAIL"

                test_data = {
                    'status': status,
                    'date': start_time.strftime('%Y-%m-%d'),
                    'ocp_version': ocp_version,
                    'test_duration': duration_str,
                    'total_runs': len(pass_rates),
                    'avg_pass_rate': f"{avg_pass_rate:.2f}%",
                    'min_pass_rate': f"{min_pass_rate_value:.2f}%",
                    'max_pass_rate': f"{max_pass_rate_value:.2f}%",
                    'total_tests': total_tests,
                    'total_failures': total_failures,
                    'skipped': total_skipped,
                    'failed_tests_summary': failed_tests_summary,
                    'notes': f"Threshold: {args.min_pass_rate}%, Started: {start_time.strftime('%Y-%m-%d %H:%M:%S')}"
                }

                log("=" * 60, args.logfile)
                log("Writing test results to Google Sheets...", args.logfile)
                # Mark row as failed (red background) if avg_pass_rate < min_pass_rate
                is_failed = (status == "FAIL")
                success = write_to_google_sheet(
                    args.google_sheet_id,
                    args.google_sheet_name,
                    args.google_credentials,
                    test_data,
                    args.logfile,
                    start_cell=args.google_sheet_start_cell,
                    mark_failed=is_failed
                )

                if success:
                    log("Successfully wrote test results to Google Sheets", args.logfile)
                    # Send Slack notification if webhook is configured
                    if args.slack_webhook:
                        slack_message = (
                            f":white_check_mark: *Long-Duration Test Completed*\n\n"
                            f"*Test Summary:*\n"
                            f"• OCP Version: {ocp_version}\n"
                            f"• Duration: {duration_str}\n"
                            f"• Total Runs: {len(pass_rates)}\n"
                            f"• Average Pass Rate: {avg_pass_rate:.2f}%\n"
                            f"• Min Pass Rate: {min_pass_rate_value:.2f}%\n"
                            f"• Max Pass Rate: {max_pass_rate_value:.2f}%\n\n"
                            f"Results written to Google Sheets successfully.\n\n"
                            f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                        )
                        send_slack_message(args.slack_webhook, slack_message, args.logfile)
                else:
                    log("Failed to write test results to Google Sheets", args.logfile)
                log("=" * 60, args.logfile)

    sys.exit(0)

if __name__ == "__main__":
    main()
