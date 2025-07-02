package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FIT file constants
const (
	FIT_HEADER_SIZE = 12
	FIT_CRC_SIZE    = 2
)

// FitHeader represents the FIT file header
type FitHeader struct {
	HeaderSize      uint8
	ProtocolVersion uint8
	ProfileVersion  uint16
	DataSize        uint32
	DataType        [4]byte
	CRC             uint16
}

// FitRecord represents a FIT data record
type FitRecord struct {
	Header    uint8
	Fields    map[uint8]interface{}
	Timestamp time.Time
}

// FitParser handles FIT file parsing
type FitParser struct {
	file   *os.File
	header FitHeader
}

// NewFitParser creates a new FIT parser
func NewFitParser(filename string) (*FitParser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	parser := &FitParser{file: file}
	if err := parser.parseHeader(); err != nil {
		file.Close()
		return nil, err
	}

	return parser, nil
}

// parseHeader parses the FIT file header
func (fp *FitParser) parseHeader() error {
	headerBytes := make([]byte, FIT_HEADER_SIZE)
	if _, err := fp.file.Read(headerBytes); err != nil {
		return err
	}

	buf := bytes.NewReader(headerBytes)

	if err := binary.Read(buf, binary.LittleEndian, &fp.header.HeaderSize); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &fp.header.ProtocolVersion); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &fp.header.ProfileVersion); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &fp.header.DataSize); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &fp.header.DataType); err != nil {
		return err
	}

	// Verify data type is ".FIT"
	if string(fp.header.DataType[:]) != ".FIT" {
		return fmt.Errorf("invalid FIT file: expected .FIT, got %s", string(fp.header.DataType[:]))
	}

	return nil
}

// ParseRecords parses all data records from the FIT file
func (fp *FitParser) ParseRecords() ([]FitRecord, error) {
	var records []FitRecord

	// Skip to data section
	if _, err := fp.file.Seek(int64(fp.header.HeaderSize), 0); err != nil {
		return nil, err
	}

	// Read data section
	dataBytes := make([]byte, fp.header.DataSize)
	if _, err := fp.file.Read(dataBytes); err != nil {
		return nil, err
	}

	// Parse records from data bytes
	// This is a simplified implementation - real FIT parsing is more complex
	offset := 0
	for offset < len(dataBytes) {
		if offset+1 >= len(dataBytes) {
			break
		}

		record := FitRecord{
			Header:    dataBytes[offset],
			Fields:    make(map[uint8]interface{}),
			Timestamp: time.Now(),
		}

		// Simple mock parsing - in reality this would decode the actual FIT protocol
		if record.Header&0x80 == 0 { // Normal header
			if offset+4 < len(dataBytes) {
				record.Fields[0] = binary.LittleEndian.Uint32(dataBytes[offset+1 : offset+5])
				offset += 5
			} else {
				break
			}
		} else { // Compressed timestamp header
			offset += 1
		}

		records = append(records, record)

		if offset+4 >= len(dataBytes) {
			break
		}
	}

	return records, nil
}

// ParseToActivity converts FIT records to Activity struct
func (fp *FitParser) ParseToActivity() (*Activity, error) {
	records, err := fp.ParseRecords()
	if err != nil {
		return nil, err
	}

	// Create activity from parsed records
	// This is a simplified conversion - real implementation would
	// decode specific FIT message types
	activity := &Activity{
		Name:      "FIT Activity",
		Type:      "unknown",
		StartTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Extract basic metrics from records
	var totalDistance float64
	var totalCalories int
	var duration int
	var maxHR, avgHR, hrCount int

	for _, record := range records {
		// Mock data extraction - real implementation would decode
		// specific FIT fields based on message type
		if val, ok := record.Fields[0]; ok {
			switch v := val.(type) {
			case uint32:
				// Simulate different field types
				fieldType := v % 10
				switch fieldType {
				case 0: // Distance (in meters, convert to km)
					totalDistance += float64(v%10000) / 1000.0
				case 1: // Calories
					totalCalories += int(v % 1000)
				case 2: // Duration (seconds)
					duration += int(v % 3600)
				case 3: // Heart rate
					hr := int(v%200) + 60 // 60-260 bpm range
					if hr > maxHR {
						maxHR = hr
					}
					avgHR += hr
					hrCount++
				}
			}
		}
	}

	activity.Distance = totalDistance
	activity.Calories = totalCalories
	activity.Duration = duration
	activity.MaxHR = maxHR
	if hrCount > 0 {
		activity.AvgHR = avgHR / hrCount
	}

	return activity, nil
}

// Close closes the FIT file
func (fp *FitParser) Close() error {
	return fp.file.Close()
}

// FitProcessor handles processing of FIT files
type FitProcessor struct {
	dataPath string
}

// NewFitProcessor creates a new FIT processor
func NewFitProcessor(dataPath string) *FitProcessor {
	return &FitProcessor{dataPath: dataPath}
}

// ProcessFitFiles processes all FIT files in the data directory
func (fp *FitProcessor) ProcessFitFiles() error {
	return filepath.Walk(fp.dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".fit" {
			fmt.Printf("Processing FIT file: %s\n", path)
			if err := fp.processSingleFitFile(path); err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				// Continue processing other files
				return nil
			}
		}

		return nil
	})
}

// processSingleFitFile processes a single FIT file
func (fp *FitProcessor) processSingleFitFile(filename string) error {
	parser, err := NewFitParser(filename)
	if err != nil {
		return err
	}
	defer parser.Close()

	activity, err := parser.ParseToActivity()
	if err != nil {
		return err
	}

	// Store activity in database
	return storeActivity(activity)
}

// storeActivity stores an activity in the database
func storeActivity(activity *Activity) error {
	query := `INSERT INTO activities 
		(name, type, start_time, duration, distance, calories, avg_hr, max_hr, elevation_gain)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		activity.Name,
		activity.Type,
		activity.StartTime,
		activity.Duration,
		activity.Distance,
		activity.Calories,
		activity.AvgHR,
		activity.MaxHR,
		activity.ElevationGain,
	)

	if err != nil {
		return fmt.Errorf("failed to store activity: %w", err)
	}

	fmt.Printf("Stored activity: %s (%.2f km, %d cal)\n",
		activity.Name, activity.Distance, activity.Calories)
	return nil
}
