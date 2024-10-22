package main

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	Position mgl32.Vec3
	Front    mgl32.Vec3
	Up       mgl32.Vec3
	Right    mgl32.Vec3
	WorldUp  mgl32.Vec3

	Yaw   float32
	Pitch float32

	MovementSpeed float32
	MouseSens     float32

	firstMouse bool
	lastX      float64
	lastY      float64
}

func NewCamera() *Camera {
	cam := &Camera{
		Position:      mgl32.Vec3{0, 0, 3},
		Front:         mgl32.Vec3{0, 0, -1},
		WorldUp:       mgl32.Vec3{0, 1, 0},
		Yaw:           -90,
		Pitch:         0,
		MovementSpeed: 2.5,
		MouseSens:     0.1,
		firstMouse:    true,
	}
	cam.updateCameraVectors()
	return cam
}

func (c *Camera) updateCameraVectors() {
	front := mgl32.Vec3{
		float32(math.Cos(float64(mgl32.DegToRad(c.Yaw))) * math.Cos(float64(mgl32.DegToRad(c.Pitch)))),
		float32(math.Sin(float64(mgl32.DegToRad(c.Pitch)))),
		float32(math.Sin(float64(mgl32.DegToRad(c.Yaw))) * math.Cos(float64(mgl32.DegToRad(c.Pitch)))),
	}
	c.Front = front.Normalize()
	c.Right = c.Front.Cross(c.WorldUp).Normalize()
	c.Up = c.Right.Cross(c.Front).Normalize()
}

func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	return mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
}

// HandleMouseMovement processes mouse input and updates camera orientation
func (c *Camera) HandleMouseMovement(window *glfw.Window, xpos, ypos float64) {
	if c.firstMouse {
		c.lastX = xpos
		c.lastY = ypos
		c.firstMouse = false
		return
	}

	xoffset := xpos - c.lastX
	yoffset := c.lastY - ypos
	c.lastX = xpos
	c.lastY = ypos

	xoffset *= float64(c.MouseSens)
	yoffset *= float64(c.MouseSens)

	c.Yaw += float32(xoffset)
	c.Pitch += float32(yoffset)

	// Constrain pitch
	if c.Pitch > 89.0 {
		c.Pitch = 89.0
	}
	if c.Pitch < -89.0 {
		c.Pitch = -89.0
	}

	c.updateCameraVectors()
}

// HandleKeyboard processes keyboard input for camera movement
func (c *Camera) HandleKeyboard(window *glfw.Window, deltaTime float64) {
	cameraSpeed := float32(deltaTime) * c.MovementSpeed

	if window.GetKey(glfw.KeyW) == glfw.Press {
		c.Position = c.Position.Add(c.Front.Mul(cameraSpeed))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		c.Position = c.Position.Sub(c.Front.Mul(cameraSpeed))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		c.Position = c.Position.Sub(c.Right.Mul(cameraSpeed))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		c.Position = c.Position.Add(c.Right.Mul(cameraSpeed))
	}
}
