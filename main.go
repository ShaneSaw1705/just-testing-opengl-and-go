package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func createShader(source string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		fmt.Printf("Failed to compile shader: %v\n", log)
	}

	return shader
}

func createProgram(vertexShaderSource, fragmentShaderSource string) uint32 {
	vertexShader := createShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := createShader(fragmentShaderSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		fmt.Printf("Failed to link program: %v\n", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program
}

func loadTexture(filename string) uint32 {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open texture file: %v", err)
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		log.Fatalf("failed to decode JPEG: %v", err)
	}

	rgba := image.NewRGBA(img.Bounds())
	for y := 0; y < rgba.Bounds().Dy(); y++ {
		for x := 0; x < rgba.Bounds().Dx(); x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Bounds().Dx()),
		int32(rgba.Bounds().Dy()),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return texture
}

type FPSCounter struct {
	lastUpdate float64
	frameCount int
	fps        float64
}

func NewFPSCounter() *FPSCounter {
	return &FPSCounter{
		lastUpdate: glfw.GetTime(),
		frameCount: 0,
		fps:        0,
	}
}

func (fc *FPSCounter) Update() float64 {
	currentTime := glfw.GetTime()
	fc.frameCount++

	// Update FPS every 0.25 seconds
	if currentTime-fc.lastUpdate >= 0.25 {
		// Calculate FPS using time difference
		fps := float64(fc.frameCount) / (currentTime - fc.lastUpdate)

		// Smooth the FPS value
		if fc.fps == 0 {
			fc.fps = fps
		} else {
			fc.fps = fc.fps*0.9 + fps*0.1 // Exponential moving average
		}

		fc.frameCount = 0
		fc.lastUpdate = currentTime
	}

	return fc.fps
}

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatal("failed to initialize GLFW:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(800, 600, "opengl go test", nil, nil)
	if err != nil {
		log.Fatal("failed to create GLFW window:", err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		log.Fatal("failed to initialize OpenGL:", err)
	}

	// Enable depth testing
	gl.Enable(gl.DEPTH_TEST)

	// Create camera
	camera := NewCamera()
	camera.Position = mgl32.Vec3{0, 0, 15} // Move camera back to see the grid

	// Capture cursor
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	// Set callbacks
	window.SetCursorPosCallback(func(w *glfw.Window, xpos float64, ypos float64) {
		if camera.firstMouse {
			camera.lastX = xpos
			camera.lastY = ypos
			camera.firstMouse = false
		}

		xoffset := xpos - camera.lastX
		yoffset := camera.lastY - ypos
		camera.lastX = xpos
		camera.lastY = ypos

		xoffset *= float64(camera.MouseSens)
		yoffset *= float64(camera.MouseSens)

		camera.Yaw += float32(xoffset)
		camera.Pitch += float32(yoffset)

		if camera.Pitch > 89.0 {
			camera.Pitch = 89.0
		}
		if camera.Pitch < -89.0 {
			camera.Pitch = -89.0
		}

		camera.updateCameraVectors()
	})

	vertexShaderSource := `
		#version 410
		layout(location = 0) in vec3 position;
		layout(location = 1) in vec2 texCoords;
		out vec2 TexCoords;
		uniform mat4 model;
		uniform mat4 view;
		uniform mat4 projection;
		void main() {
			gl_Position = projection * view * model * vec4(position, 1.0);
			TexCoords = texCoords;
		}
	` + "\x00"

	fragmentShaderSource := `
		#version 410
		in vec2 TexCoords;
		out vec4 color;
		uniform sampler2D texture1;
		void main() {
			color = texture(texture1, TexCoords);
		}
	` + "\x00"

	program := createProgram(vertexShaderSource, fragmentShaderSource)

	// Get uniform locations
	modelLoc := gl.GetUniformLocation(program, gl.Str("model\x00"))
	viewLoc := gl.GetUniformLocation(program, gl.Str("view\x00"))
	projLoc := gl.GetUniformLocation(program, gl.Str("projection\x00"))

	// Define vertices for a quad (made smaller to accommodate gaps)
	quadSize := float32(0.8) // Slightly smaller than 1.0 to create gaps
	vertices := []float32{
		// Positions          // Texture coords
		-quadSize / 2, quadSize / 2, 0.0, 0.0, 0.0, // Top left
		-quadSize / 2, -quadSize / 2, 0.0, 0.0, 1.0, // Bottom left
		quadSize / 2, -quadSize / 2, 0.0, 1.0, 1.0, // Bottom right
		quadSize / 2, quadSize / 2, 0.0, 1.0, 0.0, // Top right
	}

	indices := []uint32{
		0, 1, 2, // First triangle
		0, 2, 3, // Second triangle
	}

	var vao, vbo, ebo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.GenBuffers(1, &ebo)

	gl.BindVertexArray(vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	stride := int32(5 * unsafe.Sizeof(float32(0)))
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, nil)
	gl.EnableVertexAttribArray(0)

	texOffset := unsafe.Pointer(uintptr(3 * unsafe.Sizeof(float32(0))))
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, texOffset)
	gl.EnableVertexAttribArray(1)

	texture := loadTexture("test_grass.jpg")

	// Timing variables
	var deltaTime float64
	var lastFrame float64

	fpsCounter := NewFPSCounter()

	// Main render loop
	for !window.ShouldClose() {
		currentFrame := glfw.GetTime()
		deltaTime = currentFrame - lastFrame
		lastFrame = currentFrame

		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}

		fps := fpsCounter.Update()
		window.SetTitle(fmt.Sprintf("10x10 Grid Example | FPS: %.1f", fps))

		// Camera movement
		cameraSpeed := float32(deltaTime) * camera.MovementSpeed
		if window.GetKey(glfw.KeyW) == glfw.Press {
			camera.Position = camera.Position.Add(camera.Front.Mul(cameraSpeed))
		}
		if window.GetKey(glfw.KeyS) == glfw.Press {
			camera.Position = camera.Position.Sub(camera.Front.Mul(cameraSpeed))
		}
		if window.GetKey(glfw.KeyA) == glfw.Press {
			camera.Position = camera.Position.Sub(camera.Right.Mul(cameraSpeed))
		}
		if window.GetKey(glfw.KeyD) == glfw.Press {
			camera.Position = camera.Position.Add(camera.Right.Mul(cameraSpeed))
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(program)

		// Set up view and projection matrices
		projection := mgl32.Perspective(mgl32.DegToRad(45.0), 800.0/600.0, 0.1, 100.0)
		view := camera.GetViewMatrix()

		gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

		// Bind texture
		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.BindVertexArray(vao)

		// Draw 10x10 grid of quads
		for row := 0; row < 10; row++ {
			for col := 0; col < 10; col++ {
				// Calculate position with spacing
				xPos := float32(col) - 4.5 // Center the grid (10-1)/2 = 4.5
				yPos := float32(row) - 4.5

				// Create model matrix for this quad
				model := mgl32.Ident4()
				model = model.Mul4(mgl32.Translate3D(xPos, yPos, 0))

				// Send model matrix to shader
				gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

				// Draw the quad
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, nil)
			}
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
