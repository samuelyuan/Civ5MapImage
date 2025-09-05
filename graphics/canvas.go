package graphics

import (
	"fmt"
	"image"

	"github.com/fogleman/gg"
)

// Canvas represents an abstract drawing surface
// This allows us to decouple business logic from specific graphics libraries
type Canvas interface {
	// Basic drawing operations
	DrawRegularPolygon(sides int, x, y, radius, rotation float64)
	DrawRectangle(x, y, width, height float64)
	DrawLine(x1, y1, x2, y2 float64)

	// Color and styling
	SetColor(r, g, b uint8)
	SetLineWidth(width float64)

	// Fill and stroke operations
	Fill()
	Stroke()

	// Transformations
	InvertY()

	// Canvas management
	Resize(width, height int)

	// Text operations
	DrawString(text string, x, y float64)

	// Final output
	Image() image.Image
	SavePNG(filename string) error
}

// DrawingContext wraps the gg.Context to implement our Canvas interface
type DrawingContext struct {
	dc *gg.Context
}

// NewDrawingContext creates a new drawing context with the specified dimensions
func NewDrawingContext(width, height int) *DrawingContext {
	return &DrawingContext{
		dc: gg.NewContext(width, height),
	}
}

// Implement Canvas interface methods
func (d *DrawingContext) DrawRegularPolygon(sides int, x, y, radius, rotation float64) {
	d.dc.DrawRegularPolygon(sides, x, y, radius, rotation)
}

func (d *DrawingContext) DrawRectangle(x, y, width, height float64) {
	d.dc.DrawRectangle(x, y, width, height)
}

func (d *DrawingContext) DrawLine(x1, y1, x2, y2 float64) {
	d.dc.DrawLine(x1, y1, x2, y2)
}

func (d *DrawingContext) SetColor(r, g, b uint8) {
	d.dc.SetRGB255(int(r), int(g), int(b))
}

func (d *DrawingContext) SetLineWidth(width float64) {
	d.dc.SetLineWidth(width)
}

func (d *DrawingContext) Fill() {
	d.dc.Fill()
}

func (d *DrawingContext) Stroke() {
	d.dc.Stroke()
}

func (d *DrawingContext) InvertY() {
	d.dc.InvertY()
}

func (d *DrawingContext) Resize(width, height int) {
	d.dc = gg.NewContext(width, height)
}

func (d *DrawingContext) DrawString(text string, x, y float64) {
	d.dc.DrawString(text, x, y)
}

func (d *DrawingContext) Image() image.Image {
	return d.dc.Image()
}

func (d *DrawingContext) SavePNG(filename string) error {
	return gg.SavePNG(filename, d.dc.Image())
}

// MockCanvas for testing - implements Canvas interface without actual drawing
type MockCanvas struct {
	operations    []string
	width, height int
}

func NewMockCanvas(width, height int) *MockCanvas {
	return &MockCanvas{
		operations: make([]string, 0),
		width:      width,
		height:     height,
	}
}

func (m *MockCanvas) DrawRegularPolygon(sides int, x, y, radius, rotation float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawRegularPolygon(%d, %.2f, %.2f, %.2f, %.2f)", sides, x, y, radius, rotation))
}

func (m *MockCanvas) DrawRectangle(x, y, width, height float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawRectangle(%.2f, %.2f, %.2f, %.2f)", x, y, width, height))
}

func (m *MockCanvas) DrawLine(x1, y1, x2, y2 float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawLine(%.2f, %.2f, %.2f, %.2f)", x1, y1, x2, y2))
}

func (m *MockCanvas) SetColor(r, g, b uint8) {
	m.operations = append(m.operations,
		fmt.Sprintf("SetColor(%d, %d, %d)", r, g, b))
}

func (m *MockCanvas) SetLineWidth(width float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("SetLineWidth(%.2f)", width))
}

func (m *MockCanvas) Fill() {
	m.operations = append(m.operations, "Fill()")
}

func (m *MockCanvas) Stroke() {
	m.operations = append(m.operations, "Stroke()")
}

func (m *MockCanvas) InvertY() {
	m.operations = append(m.operations, "InvertY()")
}

func (m *MockCanvas) Resize(width, height int) {
	m.operations = append(m.operations,
		fmt.Sprintf("Resize(%d, %d)", width, height))
	m.width = width
	m.height = height
}

func (m *MockCanvas) DrawString(text string, x, y float64) {
	m.operations = append(m.operations,
		fmt.Sprintf("DrawString(\"%s\", %.2f, %.2f)", text, x, y))
}

func (m *MockCanvas) Image() image.Image {
	// Return a simple 1x1 image for testing
	return image.NewRGBA(image.Rect(0, 0, 1, 1))
}

func (m *MockCanvas) SavePNG(filename string) error {
	m.operations = append(m.operations,
		fmt.Sprintf("SavePNG(\"%s\")", filename))
	return nil
}

// GetOperations returns the list of drawing operations for testing
func (m *MockCanvas) GetOperations() []string {
	return m.operations
}
