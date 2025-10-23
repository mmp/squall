#!/usr/bin/env python3
"""
test_random_hrrr.py - Downloads random GRIB2 files from NOAA HRRR GCS bucket
and validates squall's decoding against wgrib2.

This script:
1. Randomly selects GRIB2 files from the High-Resolution Rapid Refresh (HRRR) dataset
2. Downloads them from Google Cloud Storage
3. Runs squall's integration tests to compare against wgrib2
4. Reports detailed results including value comparisons

Usage:
    python3 test_random_hrrr.py [--num-files N] [--keep-files] [--max-size MB]

Arguments:
    --num-files N    Number of random files to test (default: 5)
    --keep-files     Keep downloaded files after testing (default: clean up)
    --max-size MB    Maximum file size in MB to download (default: 50)
    --verbose        Show detailed output from tests

Requirements:
    - gsutil (Google Cloud SDK) or gcloud storage
    - wgrib2
    - Go (for running squall tests)

Examples:
    python3 test_random_hrrr.py                      # Test 5 random files
    python3 test_random_hrrr.py --num-files 10       # Test 10 random files
    python3 test_random_hrrr.py --keep-files         # Keep downloaded files
"""

import argparse
import datetime
import os
import random
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import List, Tuple, Optional


class Colors:
    """ANSI color codes for terminal output."""
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    MAGENTA = '\033[0;35m'
    CYAN = '\033[0;36m'
    BOLD = '\033[1m'
    NC = '\033[0m'  # No Color


def check_dependencies() -> bool:
    """Check if required dependencies are installed."""
    print(f"{Colors.BLUE}Checking dependencies...{Colors.NC}")

    dependencies = {
        'gsutil': 'Google Cloud SDK - https://cloud.google.com/sdk/docs/install',
        'wgrib2': 'wgrib2 - http://www.cpc.ncep.noaa.gov/products/wesley/wgrib2/',
        'go': 'Go - https://golang.org/doc/install',
    }

    missing = []
    for cmd, install_info in dependencies.items():
        if not shutil.which(cmd):
            print(f"{Colors.RED}✗ {cmd} not found{Colors.NC}")
            print(f"  Install: {install_info}")
            missing.append(cmd)
        else:
            version = get_version(cmd)
            print(f"{Colors.GREEN}✓ {cmd} found{Colors.NC} {version}")

    if missing:
        print(f"\n{Colors.RED}Missing dependencies: {', '.join(missing)}{Colors.NC}")
        return False

    print(f"{Colors.GREEN}All dependencies found.{Colors.NC}\n")
    return True


def get_version(cmd: str) -> str:
    """Get version string for a command."""
    try:
        if cmd == 'wgrib2':
            result = subprocess.run([cmd, '-version'], capture_output=True, text=True, timeout=5)
            match = re.search(r'v(\d+\.\d+\.\d+[^ ]*)', result.stdout)
            return f"({match.group(1)})" if match else ""
        elif cmd == 'go':
            result = subprocess.run([cmd, 'version'], capture_output=True, text=True, timeout=5)
            match = re.search(r'go(\d+\.\d+(?:\.\d+)?)', result.stdout)
            return f"({match.group(1)})" if match else ""
        elif cmd == 'gsutil':
            result = subprocess.run([cmd, 'version'], capture_output=True, text=True, timeout=5)
            match = re.search(r'(\d+\.\d+)', result.stdout)
            return f"({match.group(1)})" if match else ""
    except Exception:
        pass
    return ""


def get_available_dates(days_back: int = 365 * 11) -> List[str]:
    """Get list of available dates from HRRR bucket, prioritizing recent dates."""
    print(f"{Colors.BLUE}Generating date range from HRRR archive...{Colors.NC}")

    # For reliability and speed, focus on recent dates where data definitely exists
    # HRRR keeps approximately last 7-10 days of data readily available
    end_date = datetime.datetime.now() - datetime.timedelta(days=1)  # Yesterday (today might be incomplete)
    start_date = end_date - datetime.timedelta(days=10)  # Last 10 days

    dates = []
    current = start_date
    while current <= end_date:
        dates.append(current.strftime('%Y%m%d'))
        current += datetime.timedelta(days=1)

    # Shuffle for randomness
    random.shuffle(dates)

    print(f"{Colors.GREEN}Generated {len(dates)} recent dates (last 10 days){Colors.NC}")
    return dates


def get_random_files_from_date(date: str, count: int, max_size_mb: Optional[float], verbose: bool = False) -> List[Tuple[str, int]]:
    """Get random GRIB2 files from a specific date with their sizes."""
    if verbose:
        print(f"{Colors.BLUE}Trying date {date}...{Colors.NC}", end=' ')

    bucket = "gs://high-resolution-rapid-refresh"
    path = f"{bucket}/hrrr.{date}/conus/"

    try:
        # Use -l to get sizes (shorter timeout to fail fast on missing dates)
        result = subprocess.run(
            ['gsutil', 'ls', '-l', path],
            capture_output=True,
            text=True,
            timeout=15
        )

        if result.returncode != 0:
            if verbose:
                print(f"{Colors.YELLOW}not found{Colors.NC}")
            return []

        # Parse output: size date time url
        files = []
        max_size_bytes = max_size_mb * 1024 * 1024 if max_size_mb else float('inf')

        for line in result.stdout.splitlines():
            parts = line.split()
            if len(parts) >= 3 and parts[-1].endswith('.grib2') and '.wrfnatf' in parts[-1]:
                try:
                    size = int(parts[0])
                    url = parts[-1]
                    if size <= max_size_bytes:
                        files.append((url, size))
                except ValueError:
                    continue

        # Shuffle and return requested count
        random.shuffle(files)
        selected = files[:count]

        if verbose:
            if selected:
                print(f"{Colors.GREEN}found {len(selected)} file(s){Colors.NC}")
            else:
                print(f"{Colors.YELLOW}no files within size limit{Colors.NC}")

        return selected

    except subprocess.TimeoutExpired:
        if verbose:
            print(f"{Colors.YELLOW}timeout{Colors.NC}")
        return []
    except Exception as e:
        if verbose:
            print(f"{Colors.YELLOW}error: {e}{Colors.NC}")
        return []


def download_file(gcs_path: str, local_path: Path) -> bool:
    """Download a GRIB2 file from GCS."""
    filename = os.path.basename(gcs_path)
    print(f"{Colors.BLUE}Downloading: {filename}{Colors.NC}")

    try:
        result = subprocess.run(
            ['gsutil', '-q', 'cp', gcs_path, str(local_path)],
            capture_output=True,
            timeout=300  # 5 minute timeout
        )

        if result.returncode == 0 and local_path.exists():
            size_mb = local_path.stat().st_size / (1024 * 1024)
            print(f"{Colors.GREEN}✓ Downloaded: {filename} ({size_mb:.1f} MB){Colors.NC}")
            return True
        else:
            print(f"{Colors.YELLOW}✗ Failed to download {filename}{Colors.NC}")
            return False

    except subprocess.TimeoutExpired:
        print(f"{Colors.YELLOW}✗ Timeout downloading {filename}{Colors.NC}")
        return False
    except Exception as e:
        print(f"{Colors.YELLOW}✗ Error downloading {filename}: {e}{Colors.NC}")
        return False


def test_file(file_path: Path, verbose: bool = False) -> bool:
    """Run integration test on a file."""
    basename = file_path.name

    print(f"\n{Colors.BLUE}{'=' * 60}{Colors.NC}")
    print(f"{Colors.BOLD}Testing: {basename}{Colors.NC}")
    print(f"{Colors.BLUE}{'=' * 60}{Colors.NC}")

    try:
        # Run Go integration test
        cmd = [
            'go', 'test',
            '-v',
            '-timeout=10m',
            f'-run=TestIntegrationWithRealFiles/{re.escape(basename)}',
            '-no-size-limit'
        ]

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=600  # 10 minute timeout
        )

        # Print output
        if verbose or result.returncode != 0:
            print(result.stdout)
            if result.stderr:
                print(result.stderr)

        if result.returncode == 0:
            print(f"{Colors.GREEN}✓ PASSED: {basename}{Colors.NC}")
            return True
        else:
            print(f"{Colors.RED}✗ FAILED: {basename}{Colors.NC}")
            return False

    except subprocess.TimeoutExpired:
        print(f"{Colors.RED}✗ TIMEOUT: {basename}{Colors.NC}")
        return False
    except Exception as e:
        print(f"{Colors.RED}✗ ERROR testing {basename}: {e}{Colors.NC}")
        return False


def main():
    """Main function."""
    parser = argparse.ArgumentParser(
        description='Test squall against random HRRR GRIB2 files',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    parser.add_argument('--num-files', type=int, default=5,
                        help='Number of random files to test (default: 5)')
    parser.add_argument('--keep-files', action='store_true',
                        help='Keep downloaded files after testing')
    parser.add_argument('--max-size', type=float, default=None,
                        help='Maximum file size in MB to download (default: no limit)')
    parser.add_argument('--verbose', action='store_true',
                        help='Show detailed test output')

    args = parser.parse_args()

    # Print header
    print(f"{Colors.BLUE}{'═' * 60}{Colors.NC}")
    print(f"{Colors.BOLD}  HRRR Random GRIB2 Testing Script{Colors.NC}")
    print(f"{Colors.BLUE}  Testing squall vs wgrib2 on random HRRR files{Colors.NC}")
    print(f"{Colors.BLUE}{'═' * 60}{Colors.NC}\n")

    print(f"Testing {Colors.GREEN}{args.num_files}{Colors.NC} random GRIB2 files from HRRR dataset")
    if args.max_size:
        print(f"Max file size: {Colors.GREEN}{args.max_size}{Colors.NC} MB\n")
    else:
        print(f"Max file size: {Colors.GREEN}no limit{Colors.NC}\n")

    # Check dependencies
    if not check_dependencies():
        return 1

    # Create temp directory
    testgribs_dir = Path('testgribs')
    testgribs_dir.mkdir(exist_ok=True)

    temp_dir = testgribs_dir / f"hrrr_random_{datetime.datetime.now().strftime('%Y%m%d_%H%M%S')}"
    temp_dir.mkdir(exist_ok=True)

    print(f"Output directory: {Colors.CYAN}{temp_dir}{Colors.NC}\n")

    try:
        # Get available dates (all available from 2014 to present)
        dates = get_available_dates()
        if not dates:
            print(f"{Colors.RED}Error: Could not generate date range.{Colors.NC}")
            return 1

        # Collect random files from different dates
        # Keep trying dates until we get enough files
        if args.max_size:
            print(f"\n{Colors.BLUE}Searching for {args.num_files} files (max {args.max_size} MB each)...{Colors.NC}")
        else:
            print(f"\n{Colors.BLUE}Searching for {args.num_files} files (no size limit)...{Colors.NC}")

        files_to_download = []
        dates_tried = 0
        dates_with_files = 0

        for date in dates:
            if len(files_to_download) >= args.num_files:
                break

            dates_tried += 1

            # Try to get 1-2 files from this date
            date_files = get_random_files_from_date(date, 2, args.max_size, verbose=args.verbose)

            if date_files:
                dates_with_files += 1
                files_to_download.extend(date_files)

                if not args.verbose:
                    # Show progress dots
                    if dates_with_files % 5 == 0:
                        print(f"  Found {len(files_to_download)} files so far (tried {dates_tried} dates)...")

            # Safety limit: don't try forever
            if dates_tried >= 20:
                print(f"\n{Colors.YELLOW}Tried {dates_tried} dates, stopping search.{Colors.NC}")
                break

        # Trim to requested number
        files_to_download = files_to_download[:args.num_files]

        if not files_to_download:
            print(f"{Colors.RED}Error: No GRIB2 files found.{Colors.NC}")
            if args.max_size:
                print(f"{Colors.YELLOW}Try increasing --max-size{Colors.NC}")
            return 1

        print(f"\n{Colors.GREEN}Selected {len(files_to_download)} files for testing{Colors.NC}")
        print(f"  (Searched {dates_tried} dates, found files in {dates_with_files} dates)")

        # Download files
        print(f"\n{Colors.BLUE}Downloading files...{Colors.NC}")
        downloaded = []

        for gcs_path, size in files_to_download:
            filename = os.path.basename(gcs_path)
            local_path = temp_dir / filename

            if download_file(gcs_path, local_path):
                downloaded.append(local_path)

        if not downloaded:
            print(f"{Colors.RED}Error: No files were successfully downloaded.{Colors.NC}")
            return 1

        print(f"\n{Colors.GREEN}Successfully downloaded {len(downloaded)} files{Colors.NC}")

        # Test each file
        print(f"\n{Colors.BLUE}Running integration tests...{Colors.NC}")
        passed = 0
        failed = 0
        failed_files = []

        for file_path in downloaded:
            if test_file(file_path, verbose=args.verbose):
                passed += 1
            else:
                failed += 1
                failed_files.append(file_path.name)

        # Summary
        print(f"\n{Colors.BLUE}{'═' * 60}{Colors.NC}")
        print(f"{Colors.BOLD}  Test Summary{Colors.NC}")
        print(f"{Colors.BLUE}{'═' * 60}{Colors.NC}\n")

        print(f"Total files tested: {len(downloaded)}")
        print(f"{Colors.GREEN}Passed: {passed}{Colors.NC}")

        if failed > 0:
            print(f"{Colors.RED}Failed: {failed}{Colors.NC}")
            print(f"\n{Colors.RED}Failed files:{Colors.NC}")
            for filename in failed_files:
                print(f"  {Colors.RED}✗{Colors.NC} {filename}")

        # Cleanup or keep files
        if args.keep_files:
            print(f"\nTest files kept at: {Colors.CYAN}{temp_dir}{Colors.NC}")
        else:
            print(f"\n{Colors.YELLOW}Cleaning up downloaded files...{Colors.NC}")
            shutil.rmtree(temp_dir)
            print(f"{Colors.GREEN}Cleanup complete.{Colors.NC}")

        # Exit with appropriate code
        if failed > 0:
            return 1

        print(f"\n{Colors.GREEN}{Colors.BOLD}All tests passed! ✓{Colors.NC}")
        return 0

    except KeyboardInterrupt:
        print(f"\n{Colors.YELLOW}Interrupted by user{Colors.NC}")
        return 130
    except Exception as e:
        print(f"\n{Colors.RED}Unexpected error: {e}{Colors.NC}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == '__main__':
    sys.exit(main())
