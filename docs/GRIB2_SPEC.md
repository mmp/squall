# GRIB2 Format Specification Summary

## Overview

GRIB2 (GRIdded Binary 2nd edition) is a WMO (World Meteorological Organization) standard format for storing and exchanging gridded meteorological data. It is the successor to GRIB1 and provides better compression, more metadata, and support for complex data types.

**Key Characteristics:**
- Binary format optimized for compression
- Self-describing (contains metadata about the data)
- Supports multiple grid types (lat/lon, Gaussian, Lambert, etc.)
- Supports multiple compression methods (simple packing, JPEG2000, PNG)
- Extensible via templates

**File Extension:** `.grib2`, `.grb2`, or `.grib`

**Byte Order:** Big-endian (network byte order)

## Message Structure

A GRIB2 file contains one or more **messages** (also called **records**). Each message consists of 8 sections:

```
┌──────────────────────────────────────────────┐
│ Section 0: Indicator Section                 │ (16 bytes fixed)
├──────────────────────────────────────────────┤
│ Section 1: Identification Section            │ (variable length)
├──────────────────────────────────────────────┤
│ Section 2: Local Use Section (optional)      │ (variable length)
├──────────────────────────────────────────────┤
│ Section 3: Grid Definition Section           │ (variable length) ┐
├──────────────────────────────────────────────┤                   │
│ Section 4: Product Definition Section        │ (variable length) │ Can
├──────────────────────────────────────────────┤                   │ repeat
│ Section 5: Data Representation Section       │ (variable length) │ for
├──────────────────────────────────────────────┤                   │ multi-
│ Section 6: Bit Map Section                   │ (variable length) │ field
├──────────────────────────────────────────────┤                   │ msgs
│ Section 7: Data Section                      │ (variable length) ┘
├──────────────────────────────────────────────┤
│ Section 8: End Section                       │ ("7777", 4 bytes)
└──────────────────────────────────────────────┘
```

**Note:** Sections 2-7 can repeat within a single message. Section 8 always marks the end.

## Section Details

### Section 0: Indicator Section (16 bytes)

Identifies the file as GRIB2 and provides basic message info.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | "GRIB" (magic number, 0x47524942)
5-6   | 2      | Reserved (0x0000)
7     | 1      | Discipline (Table 0.0)
8     | 1      | GRIB edition number (2 for GRIB2)
9-16  | 8      | Total length of GRIB message in bytes (uint64)
```

**Discipline (Table 0.0):**
- 0 = Meteorological products
- 1 = Hydrological products
- 2 = Land surface products
- 3 = Space products
- 4 = Space weather products
- 10 = Oceanographic products
- 20 = Health and socioeconomic impacts

**Validation:**
1. Check bytes 1-4 == "GRIB"
2. Check byte 8 == 2
3. Use bytes 9-16 to determine message length

### Section 1: Identification Section

Provides metadata about the message origin and time.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (1)
6-7   | 2      | Identification of originating center (Table Common C-1)
8-9   | 2      | Identification of originating sub-center
10    | 1      | GRIB master tables version number
11    | 1      | GRIB local tables version number
12    | 1      | Significance of reference time (Table 1.2)
13-14 | 2      | Year (4 digits)
15    | 1      | Month
16    | 1      | Day
17    | 1      | Hour
18    | 1      | Minute
19    | 1      | Second
20    | 1      | Production status of data (Table 1.3)
21    | 1      | Type of processed data (Table 1.4)
```

**Originating Centers (Common C-1):**
- 7 = US NCEP
- 9 = US NWS
- 34 = Japanese Meteorological Agency
- 52 = US Air Force
- 59 = US NOAA/FSL
- 98 = ECMWF (European Centre)

**Significance of Reference Time (Table 1.2):**
- 0 = Analysis
- 1 = Start of forecast
- 2 = Verifying time of forecast
- 3 = Observation time

### Section 2: Local Use Section (Optional)

Reserved for local or experimental use. Structure is center-specific.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (2)
6-n   | n-5    | Local use data
```

Most implementations ignore this section or store it as opaque bytes.

### Section 3: Grid Definition Section

Describes the geographic grid on which the data is defined.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (3)
6     | 1      | Source of grid definition (Table 3.0)
7-10  | 4      | Number of data points (uint32)
11    | 1      | Number of octets for optional list (0 if none)
12    | 1      | Interpretation of optional list
13-14 | 2      | Grid definition template number (Table 3.1)
15-n  | n-14   | Grid definition (template-specific)
```

**Common Grid Definition Templates (Table 3.1):**

**Template 3.0: Latitude/Longitude (Equidistant Cylindrical)**
- Most common grid type
- Regular spacing in lat and lon
- Fields: Ni (# points in x), Nj (# points in y), La1, Lo1, La2, Lo2, Di, Dj, scanning mode

**Template 3.40: Gaussian Latitude/Longitude**
- Used by ECMWF and other global models
- Irregular latitude spacing (Gaussian quadrature points)
- Fields: Similar to 3.0 but with N (number of latitude circles)

**Template 3.30: Lambert Conformal**
- Conformal conic projection
- Used for regional models (e.g., NAM)
- Fields: LaD, LoV (orientation), Latin1, Latin2

**Template 3.10: Mercator**
- Cylindrical projection
- Fields: La1, Lo1, La2, Lo2, LaD, Latin

**Template 3.20: Polar Stereographic**
- Azimuthal projection
- Used for polar regions
- Fields: LaD, LoV, orientation

**Scanning Mode Flags (bit 5 of scanning mode byte):**
- Bit 0: 0=+i direction (west to east), 1=-i direction (east to west)
- Bit 1: 0=−j direction (north to south), 1=+j direction (south to north)
- Bit 2: 0=adjacent points in i direction, 1=adjacent points in j direction

### Section 4: Product Definition Section

Describes what meteorological parameter is contained in the data.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (4)
6-7   | 2      | Number of coordinate values after template (0 if none)
8-9   | 2      | Product definition template number (Table 4.0)
10-n  | n-9    | Product definition (template-specific)
```

**Common Product Definition Templates (Table 4.0):**

**Template 4.0: Analysis or Forecast at a Horizontal Level or Layer**
- Most common template
- Fields include:
  - Parameter category (Table 4.1)
  - Parameter number (Table 4.2.X.Y - depends on discipline and category)
  - Generating process (analysis, forecast, etc.)
  - Forecast time in units (hours, days, etc.)
  - Type of first fixed surface (Table 4.5)
  - Scale factor and value of first fixed surface
  - Type of second fixed surface (0xFF if not used)
  - Scale factor and value of second fixed surface

**Template 4.1: Individual Ensemble Forecast**
- Extends 4.0 with ensemble member information
- Additional fields: ensemble type, perturbation number, total ensemble members

**Template 4.8: Average, Accumulation, or Statistical Process**
- For accumulated precipitation, max/min temperature, etc.
- Includes time range information

**Parameter Identification:**
- Discipline (from Section 0)
- Parameter category (from Section 4)
- Parameter number (from Section 4)
- Together form: Discipline.Category.Number (e.g., 0.0.0 = Temperature)

**Common Parameters (Discipline 0 = Meteorological):**
- Category 0 (Temperature): 0=Temperature, 4=Maximum temp, 5=Minimum temp
- Category 1 (Moisture): 0=Specific humidity, 1=Relative humidity, 8=Total precipitation
- Category 2 (Momentum): 0=Wind direction, 1=Wind speed, 2=u-component, 3=v-component
- Category 3 (Mass): 0=Pressure, 1=Pressure reduced to MSL, 5=Geopotential height

**Level Types (Table 4.5):**
- 1 = Surface (ground or water)
- 2 = Cloud base level
- 3 = Cloud top level
- 100 = Isobaric surface (value in Pa)
- 103 = Height above ground (value in m)
- 104 = Sigma level
- 105 = Hybrid level

### Section 5: Data Representation Section

Describes how the data values are packed/compressed.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (5)
6-9   | 4      | Number of data values (uint32)
10-11 | 2      | Data representation template number (Table 5.0)
12-n  | n-11   | Data representation (template-specific)
```

**Common Data Representation Templates (Table 5.0):**

**Template 5.0: Simple Packing**
- Most common (used in 80%+ of files)
- Linear scaling: `value = (R + X * 2^E) / 10^D`
  - R = reference value (4-byte IEEE float)
  - E = binary scale factor (2-byte signed int)
  - D = decimal scale factor (2-byte signed int)
  - X = packed value (n-bit unsigned int)
- Fields:
  - Reference value (R)
  - Binary scale factor (E)
  - Decimal scale factor (D)
  - Number of bits per value (n)
  - Type of original field values (Table 5.1)

**Template 5.2: Complex Packing**
- Groups similar values for better compression
- Used for parameters with spatial correlation

**Template 5.3: Complex Packing with Spatial Differencing**
- First/second order spatial differencing
- Very efficient for smooth fields

**Template 5.40: JPEG 2000 Compression**
- Uses JPEG 2000 wavelet compression
- Lossy or lossless
- Requires JPEG2000 library

**Template 5.41: PNG Compression**
- Uses PNG compression (DEFLATE algorithm)
- Lossless
- Requires PNG library or can use standard library

**Template 5.200: Run Length Encoding**
- For sparse data (many undefined values)

### Section 6: Bit Map Section

Indicates which grid points have valid data (some may be undefined/missing).

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (6)
6     | 1      | Bit-map indicator (Table 6.0)
7-n   | n-6    | Bit map (if indicator = 0)
```

**Bit-map Indicator (Table 6.0):**
- 0 = Bit map applies to this product and is specified in this section
- 254 = Previously defined bit map applies to this product
- 255 = Bit map does not apply (all grid points are valid)

**Bit Map Format (when indicator = 0):**
- One bit per grid point (in scanning order)
- 1 = data value present
- 0 = data value absent/undefined
- Packed into bytes (8 bits per byte, padded with 0s if needed)

### Section 7: Data Section

Contains the actual packed data values.

**Structure:**
```
Byte  | Length | Description
------|--------|--------------------------------------------------
1-4   | 4      | Length of section in bytes
5     | 1      | Section number (7)
6-n   | n-5    | Packed data (format depends on Section 5)
```

The data is decoded according to the template specified in Section 5.

**Decoding Process (for simple packing):**
1. Read packed values as n-bit unsigned integers
2. Apply formula: `value = (R + X * 2^E) / 10^D`
3. Apply bit map (if present) to map values to grid points
4. Undefined values are typically set to 9.999e20

### Section 8: End Section (4 bytes)

Marks the end of the GRIB2 message.

**Structure:**
```
"7777" (ASCII: 0x37373737)
```

## Parsing Algorithm

### High-Level Flow

1. **Read Section 0** (16 bytes)
   - Validate "GRIB" magic number
   - Extract message length
   - Extract discipline

2. **Read Section 1**
   - Extract reference time
   - Extract originating center

3. **Read Section 2** (if present)
   - Store for local use

4. **Loop through sections 3-7** (may repeat)
   - Read section header (section number + length)
   - If section 3: Parse grid definition
   - If section 4: Parse product definition
   - If section 5: Parse data representation
   - If section 6: Parse bitmap
   - If section 7: Decode data values
   - When section 7 is complete:
     - Combine grid + product + data into a GRIB2 record
     - Apply filters (if any)
     - Add to results

5. **Read Section 8**
   - Validate "7777" end marker

6. **Move to next message** (if any bytes remain)

### Key Insights for Clean Implementation

**1. Sections 1-2 are per-message metadata**
   - Apply once to all fields in a message

**2. Sections 3-7 define a "data field"**
   - Can repeat multiple times within a message
   - Each section 7 completes one field

**3. Sections 3-6 can be reused**
   - A section can reference the previous occurrence
   - Example: "Use grid definition from previous field"

**4. Section numbers must appear in order**
   - Can't have section 5 before section 4
   - But sections can repeat

## Code Table Generation Strategy

The wgrib2 code includes `.dat` files that are essentially C switch statements. Example:

```c
case 0: string="Temperature"; break;
case 1: string="Specific humidity"; break;
// ... hundreds more
```

**Our Strategy:**
1. Parse these `.dat` files programmatically
2. Generate Go code with `map[int]string` lookups
3. Store in `tables/` package
4. Create code generator tool in `tables/codegen/`

**Benefits:**
- Easy to update when WMO releases new codes
- No manual transcription errors
- Fast lookups (map vs. switch)
- Data-driven, not code-driven

## Common File Characteristics

**Typical GRIB2 File Sizes:**
- Local model output (single time): 1-50 MB
- Global model output (single time): 100-500 MB
- Multi-time forecast package: 1-10 GB

**Typical Number of Messages per File:**
- Small: 1-10 messages
- Medium: 10-100 messages
- Large: 100-1000+ messages

**Most Common Combinations:**
- Grid: Lat/Lon (3.0) or Gaussian (3.40)
- Product: Analysis/Forecast (4.0)
- Packing: Simple (5.0)
- Bitmap: Not present (255)

**This means our MVP should prioritize:**
- Template 3.0 (lat/lon)
- Template 4.0 (analysis/forecast)
- Template 5.0 (simple packing)
- Bitmap handling (both present and absent)

## Reference Values and Endianness

**All multi-byte integers are big-endian (network byte order)**

**IEEE Floating Point:**
- Reference value in Section 5 is 4-byte IEEE 754 float (big-endian)
- Go's `math.Float32frombits(binary.BigEndian.Uint32(bytes))`

**Bit-level Packing:**
- Data values may not align to byte boundaries
- Need bit-reading utilities
- Example: 12-bit values means 2 values per 3 bytes

## Validation Checklist

For each GRIB2 file:
- [ ] Section 0: "GRIB" magic number
- [ ] Section 0: Edition number = 2
- [ ] Section 0: Message length matches actual length
- [ ] Section numbers appear in order
- [ ] Each section length matches actual section size
- [ ] Section 8: "7777" end marker
- [ ] Number of data values (Section 5) matches grid points (Section 3)
- [ ] Bitmap length matches grid points (if present)
- [ ] Template numbers are valid (exist in tables)

## Error Handling Strategy

**Recoverable Errors:**
- Unknown template numbers → return descriptive error
- Unknown parameter codes → use "Unknown (code X)"
- Truncated data → return what was successfully parsed

**Fatal Errors:**
- Invalid GRIB header → reject file
- Wrong edition number → reject message
- Corrupted section length → stop parsing message
- Missing required sections → reject message

## Testing Data Sources

**NOAA (Free):**
- https://nomads.ncep.noaa.gov/
- GFS, NAM, HRRR, RAP models
- Various grid types and products

**ECMWF (Free samples):**
- https://apps.ecmwf.int/datasets/data/
- ERA5 reanalysis
- Gaussian grids

**Test Files in wgrib2:**
- ~/wgrib2/tests/ directory (if available)

## WMO Documentation Sources

1. **Manual on Codes, Volume I.2 (WMO-No. 306)**
   - https://community.wmo.int/en/activity-areas/wis/publications/306-vI.2
   - Official specification

2. **WMO GRIB2 Code Registry**
   - https://codes.wmo.int/grib2
   - Up-to-date code tables

3. **NCEP GRIB2 Documentation**
   - https://www.nco.ncep.noaa.gov/pmb/docs/grib2/grib2_doc/
   - Practical examples and tables

4. **WMO GRIB2 GitHub**
   - https://github.com/wmo-im/GRIB2
   - Reference implementations

---

**Document Status**: Reference guide
**Last Updated**: 2025-10-18
**Version**: 1.0
