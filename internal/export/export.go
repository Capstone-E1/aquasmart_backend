package export

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/Capstone-E1/aquasmart_backend/internal/models"
	"github.com/xuri/excelize/v2"
)

// ExportService handles data export functionality
type ExportService struct{}

// NewExportService creates a new export service instance
func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportData represents data to be exported
type ExportData struct {
	SensorReadings          []models.SensorReading
	WaterQualityAssessments []models.WaterQualityStatus
	FiltrationHistory       []FiltrationRecord
	ExportMetadata          ExportMetadata
}

// FiltrationRecord represents a filtration session record
type FiltrationRecord struct {
	ID              int       `json:"id"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	FilterMode      string    `json:"filter_mode"`
	TargetVolume    float64   `json:"target_volume"`
	ProcessedVolume float64   `json:"processed_volume"`
	Progress        float64   `json:"progress"`
	Status          string    `json:"status"`
	Duration        string    `json:"duration"`
}

// ExportMetadata contains information about the export
type ExportMetadata struct {
	GeneratedAt   time.Time `json:"generated_at"`
	DateRange     string    `json:"date_range"`
	TotalReadings int       `json:"total_readings"`
	FilterModes   []string  `json:"filter_modes"`
	DeviceInfo    string    `json:"device_info"`
}

// GenerateExcel creates an Excel file with purification history
func (es *ExportService) GenerateExcel(data ExportData) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Set document properties
	f.SetDocProps(&excelize.DocProperties{
		Category:       "AquaSmart Water Purification",
		ContentStatus:  "Draft",
		Created:        data.ExportMetadata.GeneratedAt.Format(time.RFC3339),
		Creator:        "AquaSmart System",
		Description:    "Water purification history and sensor data export",
		LastModifiedBy: "AquaSmart Backend",
		Modified:       data.ExportMetadata.GeneratedAt.Format(time.RFC3339),
		Subject:        "Water Quality & Filtration History",
		Title:          "AquaSmart Purification Report",
		Version:        "1.0",
	})

	// Create Summary sheet
	es.createSummarySheet(f, data)

	// Create Sensor Data sheet
	es.createSensorDataSheet(f, data.SensorReadings)

	// Create Filtration History sheet
	es.createFiltrationHistorySheet(f, data.FiltrationHistory)

	// Create Water Quality Analysis sheet
	es.createWaterQualitySheet(f, data.WaterQualityAssessments)

	// Set active sheet to Summary
	f.SetActiveSheet(0)

	return f, nil
}

// createSummarySheet creates the summary overview sheet
func (es *ExportService) createSummarySheet(f *excelize.File, data ExportData) error {
	sheetName := "Summary"
	f.SetSheetName("Sheet1", sheetName)

	// Header styling
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title
	f.SetCellValue(sheetName, "A1", "AquaSmart Water Purification Report")
	f.MergeCell(sheetName, "A1", "D1")
	f.SetCellStyle(sheetName, "A1", "D1", headerStyle)
	f.SetRowHeight(sheetName, 1, 25)

	// Export metadata
	f.SetCellValue(sheetName, "A3", "Generated At:")
	f.SetCellValue(sheetName, "B3", data.ExportMetadata.GeneratedAt.Format("2006-01-02 15:04:05"))
	f.SetCellValue(sheetName, "A4", "Date Range:")
	f.SetCellValue(sheetName, "B4", data.ExportMetadata.DateRange)
	f.SetCellValue(sheetName, "A5", "Total Readings:")
	f.SetCellValue(sheetName, "B5", data.ExportMetadata.TotalReadings)

	// Statistics
	f.SetCellValue(sheetName, "A7", "System Statistics")
	f.SetCellStyle(sheetName, "A7", "A7", headerStyle)

	f.SetCellValue(sheetName, "A8", "Total Sensor Readings:")
	f.SetCellValue(sheetName, "B8", len(data.SensorReadings))
	f.SetCellValue(sheetName, "A9", "Filtration Sessions:")
	f.SetCellValue(sheetName, "B9", len(data.FiltrationHistory))
	f.SetCellValue(sheetName, "A10", "Quality Assessments:")
	f.SetCellValue(sheetName, "B10", len(data.WaterQualityAssessments))

	// Column widths
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "D", 15)

	return nil
}

// createSensorDataSheet creates the sensor readings sheet
func (es *ExportService) createSensorDataSheet(f *excelize.File, readings []models.SensorReading) error {
	sheetName := "Sensor Data"
	f.NewSheet(sheetName)

	// Headers
	headers := []string{"Timestamp", "Filter Mode", "Flow (L/min)", "pH", "Turbidity (NTU)", "TDS (ppm)"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Header styling
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	f.SetCellStyle(sheetName, "A1", "F1", headerStyle)

	// Data rows
	for i, reading := range readings {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), reading.Timestamp.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), reading.FilterMode)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), reading.Flow)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), reading.Ph)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), reading.Turbidity)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), reading.TDS)
	}

	// Format columns
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "F", 12)

	return nil
}

// createFiltrationHistorySheet creates the filtration sessions sheet
func (es *ExportService) createFiltrationHistorySheet(f *excelize.File, history []FiltrationRecord) error {
	sheetName := "Filtration History"
	f.NewSheet(sheetName)

	// Headers
	headers := []string{"Start Time", "End Time", "Duration", "Filter Mode", "Target Volume (L)", "Processed Volume (L)", "Progress (%)", "Status"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Header styling
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"C55A11"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	f.SetCellStyle(sheetName, "A1", "H1", headerStyle)

	// Data rows
	for i, record := range history {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), record.StartTime.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), record.EndTime.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), record.Duration)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), record.FilterMode)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), record.TargetVolume)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), record.ProcessedVolume)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("%.1f%%", record.Progress))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), record.Status)
	}

	// Format columns
	f.SetColWidth(sheetName, "A", "C", 20)
	f.SetColWidth(sheetName, "D", "H", 15)

	return nil
}

// createWaterQualitySheet creates the water quality analysis sheet
func (es *ExportService) createWaterQualitySheet(f *excelize.File, assessments []models.WaterQualityStatus) error {
	sheetName := "Water Quality"
	f.NewSheet(sheetName)

	// Headers
	headers := []string{"Timestamp", "Filter Mode", "pH Status", "Turbidity Status", "TDS Status", "Overall Quality", "pH Value", "Turbidity Value", "TDS Value"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Header styling
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"7030A0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	f.SetCellStyle(sheetName, "A1", "I1", headerStyle)

	// Data rows
	for i, assessment := range assessments {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), assessment.Timestamp.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), assessment.FilterMode)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), assessment.PhStatus)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), assessment.TurbStatus)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), assessment.TDSStatus)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), assessment.OverallQuality)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), assessment.Ph)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), assessment.Turbidity)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), assessment.TDS)
	}

	// Format columns
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "I", 15)

	return nil
}

// GenerateCSV creates CSV data for sensor readings
func (es *ExportService) GenerateCSV(readings []models.SensorReading) ([][]string, error) {
	// CSV headers
	records := [][]string{
		{"Timestamp", "Filter Mode", "Flow (L/min)", "pH", "Turbidity (NTU)", "TDS (ppm)"},
	}

	// Add data rows
	for _, reading := range readings {
		record := []string{
			reading.Timestamp.Format("2006-01-02 15:04:05"),
			string(reading.FilterMode),
			strconv.FormatFloat(reading.Flow, 'f', 2, 64),
			strconv.FormatFloat(reading.Ph, 'f', 2, 64),
			strconv.FormatFloat(reading.Turbidity, 'f', 2, 64),
			strconv.FormatFloat(reading.TDS, 'f', 1, 64),
		}
		records = append(records, record)
	}

	return records, nil
}

// WriteCSV writes CSV data to a writer
func (es *ExportService) WriteCSV(w *csv.Writer, records [][]string) error {
	return w.WriteAll(records)
}