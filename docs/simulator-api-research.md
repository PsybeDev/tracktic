# Racing Simulator API Research

## Overview

This document outlines the API interfaces and data access methods for the three target racing simulators: iRacing, Assetto Corsa Competizione (ACC), and Le Mans Ultimate (LMU).

## iRacing API Interface

### Connection Method

- **SDK**: goirsdk library (already in use)
- **Protocol**: Shared memory access via iRacing SDK
- **Polling Rate**: 60Hz (recommended), currently using 4Hz (250ms)
- **Connection Check**: `irsdk.IsSimRunning()`

### Available Data Points

Based on existing implementation and iRacing SDK documentation:

**Session Information:**

- `SessionTime` - Current session time (float64)
- `SessionTimeRemain` - Time remaining in session (float64)
- `SessionLapsRemain` - Laps remaining in session (int32)
- `SessionFlags` - Current session flags (int32)
- `SessionState` - Session state (invalid, get_in_car, warmup, parade_laps, racing, checkered, cool_down)
- `SessionType` - Session type (none, practice, qualify, race)

**Car/Player Data:**

- `PlayerCarIdx` - Player car index (int32)
- `FuelLevel` - Current fuel level (float32, in liters)
- `FuelUsePerHour` - Fuel consumption rate (float32)
- `LapLastLapTime` - Last completed lap time (float32, seconds)
- `CarIdxPosition` - Car positions array (int32[])
- `CarIdxLapDistPct` - Lap distance percentage array (float32[])
- `LapDistPct` - Player lap distance percentage (float32)

**Tire Data:**

- `LFtempCL`, `LFtempCM`, `LFtempCR` - Left front tire temps (float32)
- `RFtempCL`, `RFtempCM`, `RFtempCR` - Right front tire temps (float32)
- `LRtempCL`, `LRtempCM`, `LRtempCR` - Left rear tire temps (float32)
- `RRtempCL`, `RRtempCM`, `RRtempCR` - Right rear tire temps (float32)
- `LFwearL`, `LFwearM`, `LFwearR` - Left front tire wear (float32)
- `RFwearL`, `RFwearM`, `RFwearR` - Right front tire wear (float32)
- `LRwearL`, `LRwearM`, `LRwearR` - Left rear tire wear (float32)
- `RRwearL`, `RRwearM`, `RRwearR` - Right rear tire wear (float32)

**Pit Data:**

- `OnPitRoad` - Whether car is on pit road (bool)
- `CarIdxOnPitRoad` - Array of cars on pit road (bool[])
- `PitSvTireCompound` - Pit service tire compound (int32)
- `PitSvFuel` - Pit service fuel amount (float32)

**Opponent Data:**

- `CarIdxLap` - Current lap for each car (int32[])
- `CarIdxLapCompleted` - Last completed lap for each car (int32[])
- `CarIdxBestLapTime` - Best lap time for each car (float32[])
- `CarIdxLastLapTime` - Last lap time for each car (float32[])

### Data Update Frequency

- Real-time updates at 60Hz
- Session updates at lower frequency as needed

---

## Assetto Corsa Competizione (ACC) API Interface

### Connection Method

- **SDK**: ACC Shared Memory SDK
- **Protocol**: Shared memory access via Windows API
- **Files**: Physics.acd, Graphics.acd, Static.acd
- **Language**: C++ SDK with Go bindings needed

### Available Data Points

Based on ACC SDK documentation:

**Session Information (Graphics):**

- `session` - Current session type (AC_PRACTICE, AC_QUALIFY, AC_RACE, AC_HOTLAP, AC_TIME_ATTACK, AC_DRIFT, AC_DRAG, AC_HOTSTINT, AC_HOTLAPSUPERPOLE)
- `sessionTimeLeft` - Session time remaining (int32, milliseconds)
- `sessionType` - Session type string
- `flag` - Current session flags (AC_NO_FLAG, AC_BLUE_FLAG, AC_YELLOW_FLAG, AC_BLACK_FLAG, AC_WHITE_FLAG, AC_CHECKERED_FLAG, AC_PENALTY_FLAG)

**Car/Player Data (Physics):**

- `fuel` - Current fuel level (float32, liters)
- `rpms` - Current RPM (float32)
- `gear` - Current gear (int32)
- `speedKmh` - Speed in km/h (float32)
- `tc` - Traction control level (float32)
- `abs` - ABS level (float32)
- `turboBoost` - Turbo boost (float32)
- `airTemp` - Air temperature (float32)
- `roadTemp` - Road temperature (float32)

**Tire Data (Physics):**

- `tyrePressure[4]` - Tire pressure for each wheel (float32[])
- `tyreWear[4]` - Tire wear for each wheel (float32[])
- `tyreDirtyLevel[4]` - Tire dirt level (float32[])
- `tyreTemp[4]` - Tire temperature (float32[])
- `wheelSlip[4]` - Wheel slip (float32[])

**Lap Data (Graphics):**

- `iCurrentTime` - Current lap time (int32, milliseconds)
- `iLastTime` - Last lap time (int32, milliseconds)
- `iBestTime` - Best lap time (int32, milliseconds)
- `completedLaps` - Number of completed laps (int32)
- `position` - Current position (int32)

**Opponent Data:**

- Limited opponent data available compared to iRacing
- Position and basic timing information only

### Limitations

- No comprehensive opponent data like iRacing
- Limited pit stop information
- Requires Windows shared memory access

---

## Le Mans Ultimate (LMU) API Interface

### Connection Method

- **SDK**: rFactor 2 SDK (LMU is based on rFactor 2 engine)
- **Protocol**: Plugin-based system or shared memory
- **Files**: Similar to rFactor 2 implementation

### Available Data Points

Based on rFactor 2 SDK (LMU compatibility):

**Session Information:**

- `mSessionET` - Session elapsed time (double)
- `mEndET` - Session end time (double)
- `mCurrentET` - Current session time (double)
- `mGamePhase` - Game phase (garage, warmup, gridwalk, formation, countdown, greenFlag, fullCourseYellow, sessionStopped, sessionOver)
- `mYellowFlagState` - Yellow flag state
- `mSectorFlag[3]` - Sector flags

**Vehicle Data:**

- `mFuel` - Current fuel level (double)
- `mEngineRPM` - Engine RPM (double)
- `mEngineMaxRPM` - Maximum engine RPM (double)
- `mClutchRPM` - Clutch RPM (double)
- `mGear` - Current gear (long)
- `mSpeedometer` - Speed (double)

**Tire Data:**

- `mWheel[4].mTemperature[3]` - Tire temperatures (inside, middle, outside)
- `mWheel[4].mWear` - Tire wear level (double)
- `mWheel[4].mPressure` - Tire pressure (double)
- `mWheel[4].mRideHeight` - Ride height (double)

**Lap Data:**

- `mLapNumber` - Current lap number (long)
- `mLapStartET` - Lap start time (double)
- `mSector` - Current sector (0, 1, 2)
- `mCurrentSector1` - Current sector 1 time (double)
- `mCurrentSector2` - Current sector 2 time (double)
- `mLastLapTime` - Last lap time (double)
- `mBestLapTime` - Best lap time (double)

**Opponent Data:**

- `mNumVehicles` - Number of vehicles (long)
- Vehicle array with individual car data:
  - `mPlace` - Position (long)
  - `mLapNumber` - Lap number (long)
  - `mLapStartET` - Lap start time (double)
  - `mBestLapTime` - Best lap time (double)
  - `mLastLapTime` - Last lap time (double)

### Implementation Considerations

- May require rFactor 2 SDK adaptation
- Plugin development might be needed
- Less documentation available compared to iRacing/ACC

---

## Common Data Model Requirements

Based on the PRD requirements, we need to standardize the following data points across all simulators:

### Essential Data Points

1. **Fuel Data**

   - Current fuel level (liters)
   - Fuel usage rate (liters/hour or liters/lap)
   - Estimated laps remaining on current fuel

2. **Tire Data**

   - Tire wear percentage for each wheel
   - Tire temperature (average across compound)
   - Tire pressure

3. **Lap Data**

   - Current lap number
   - Last lap time
   - Best lap time
   - Current lap time (live)

4. **Session Data**

   - Session type (practice, qualifying, race)
   - Session time remaining
   - Session laps remaining (if applicable)
   - Race format (sprint vs endurance detection)

5. **Position Data**

   - Current position
   - Gap to car ahead/behind
   - Lap distance percentage

6. **Opponent Data**
   - Opponent positions
   - Opponent lap times
   - Opponent pit status

### Data Refresh Rates

- **High Priority (60Hz)**: Fuel, lap times, positions
- **Medium Priority (10Hz)**: Tire data, session status
- **Low Priority (1Hz)**: Opponent data, session info

## Implementation Strategy

1. **Phase 1**: Extend existing iRacing implementation
2. **Phase 2**: Implement ACC shared memory access
3. **Phase 3**: Research and implement LMU integration
4. **Phase 4**: Create unified data interface

### Technical Challenges

- **iRacing**: Extend existing implementation, optimize polling rate
- **ACC**: Implement Windows shared memory access in Go
- **LMU**: Research rFactor 2 SDK compatibility, potential plugin development
- **Unified Interface**: Standardize different data formats and update frequencies
