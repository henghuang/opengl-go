package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const windowWidth = 800
const windowHeight = 600

var lightPos = [3]float32{0, 0.25, 2}
var viewPos = [3]float32{3, 3, 3}

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "Cube", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	// Configure the vertex and fragment shaders
	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}
	programLight, err := newProgram(vertexShader, lightFragmentShader)
	if err != nil {
		panic(err)
	}
	// first
	gl.UseProgram(program)
	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/windowHeight, 0.1, 10.0)
	projectionUniform := gl.GetUniformLocation(program, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

	camera := mgl32.LookAtV(mgl32.Vec3{viewPos[0], viewPos[1], viewPos[2]}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
	cameraUniform := gl.GetUniformLocation(program, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

	model := mgl32.Ident4()
	modelUniform := gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

	// objectColor := mgl64.Vec3([3]float64{0.5, 0.5, 0.31})
	objectColorUniform := gl.GetUniformLocation(program, gl.Str("objectColor\x00"))
	gl.Uniform3f(objectColorUniform, 1, 0.5, 0.31)

	lightColorUniform := gl.GetUniformLocation(program, gl.Str("lightColor\x00"))
	gl.Uniform3f(lightColorUniform, 1, 1, 1)

	lightPosUniform := gl.GetUniformLocation(program, gl.Str("lightPos\x00"))
	gl.Uniform3f(lightPosUniform, lightPos[0], lightPos[1], lightPos[2])

	viewPosUniform := gl.GetUniformLocation(program, gl.Str("viewPos\x00"))
	gl.Uniform3f(viewPosUniform, viewPos[0], viewPos[1], viewPos[2])

	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

	//second
	gl.UseProgram(programLight)
	lightProjection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/windowHeight, 0.1, 10.0)
	lightProjectionUniform := gl.GetUniformLocation(programLight, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(lightProjectionUniform, 1, false, &lightProjection[0])

	lightCamera := mgl32.LookAtV(mgl32.Vec3{3, 3, 3}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
	lightCameraUniform := gl.GetUniformLocation(programLight, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(lightCameraUniform, 1, false, &lightCamera[0])

	lightModel := mgl32.Ident4()
	lightModelUniform := gl.GetUniformLocation(programLight, gl.Str("model\x00"))
	gl.UniformMatrix4fv(lightModelUniform, 1, false, &lightModel[0])

	lightTextureUniform := gl.GetUniformLocation(programLight, gl.Str("tex\x00"))
	gl.Uniform1i(lightTextureUniform, 1) //set bind to which texture index

	gl.BindFragDataLocation(programLight, 1, gl.Str("outputColor\x00"))

	// Load the texture
	// texture, err := newTexture("square.png")
	texture2, err := newTexture("square2.png")
	if err != nil {
		log.Fatalln(err)
	}

	// Configure the vertex data

	// the first
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)
	//设置vertexShader中的变量vert如何取值
	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(vertAttrib)
	//vertTexCoord 取点方法
	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(texCoordAttrib)
	//normal vector
	aNormalAttrib := uint32(gl.GetAttribLocation(program, gl.Str("aNormal\x00")))
	gl.VertexAttribPointer(aNormalAttrib, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(aNormalAttrib)

	// the second vao
	var lightVAO uint32
	gl.GenVertexArrays(1, &lightVAO)
	gl.BindVertexArray(lightVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	lightvertAttrib := uint32(gl.GetAttribLocation(programLight, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(lightvertAttrib)
	gl.VertexAttribPointer(lightvertAttrib, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))

	lightTexCoordAttrib := uint32(gl.GetAttribLocation(programLight, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(lightTexCoordAttrib)
	gl.VertexAttribPointer(lightTexCoordAttrib, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0, 0, 0, 1)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Update
		lightX := float32(2.0 * math.Sin(glfw.GetTime()))
		lightY := float32(-0.25)
		lightZ := float32(1.5 * math.Cos(glfw.GetTime()))

		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, texture2)

		// Render 1
		gl.UseProgram(program)
		gl.Uniform3f(lightPosUniform, lightX, lightY, lightZ)
		gl.Uniform3f(viewPosUniform, viewPos[0], viewPos[1], viewPos[2])

		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)

		gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

		// Render2
		newModel := mgl32.Translate3D(lightX, lightY, lightZ).Mul4(mgl32.Scale3D(0.2, 0.2, 0.2))
		gl.UseProgram(programLight)
		gl.UniformMatrix4fv(lightModelUniform, 1, false, &newModel[0])
		gl.BindVertexArray(lightVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

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

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
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

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newTexture(file string) (uint32, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}

var vertexShader = `
#version 330
uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
in vec3 vert;
in vec2 vertTexCoord;
in vec3 aNormal; //norm vector
out vec2 fragTexCoord;
out vec3 Normal;
out vec3 FragPos;
void main() {
    fragTexCoord = vertTexCoord;
	gl_Position = projection * camera * model * vec4(vert, 1);
	FragPos = vec3(model * vec4(vert, 1.0));
	Normal = aNormal;
}
` + "\x00"

var fragmentShader = `
#version 330
uniform vec3 objectColor;
uniform vec3 lightColor;
uniform vec3 lightPos;
uniform vec3 viewPos;
in vec3 Normal;
in vec3 FragPos;  
out vec4 outputColor;
void main() {
	vec3 norm = normalize(Normal);
	vec3 lightDir = normalize(lightPos - FragPos);  
	float diff = max(dot(norm, lightDir), 0.0);
	vec3 diffuse = diff * lightColor;

	float specularStrength = 0.5;
	vec3 viewDir = normalize(viewPos - FragPos);
	vec3 reflectDir = reflect(-lightDir, norm); 
	float spec = pow(max(dot(viewDir, reflectDir), 0.0), 256);
	vec3 specular = specularStrength * spec * lightColor;   

	float ambientStrength = 0.1;
	vec3 ambient = ambientStrength * lightColor;
	vec3 result = (ambient+ diffuse+specular) * objectColor;
	outputColor = vec4(result, 1);
}
` + "\x00"

var lightFragmentShader = `
#version 330
uniform sampler2D tex;
in vec2 fragTexCoord;
out vec4 outputColor;
void main() {
	// outputColor = vec4(1);
	outputColor = texture(tex, fragTexCoord);
}
` + "\x00"

var cubeVertices = []float32{
	//  X, Y, Z, U, V,X,Y,Z norm
	// Bottom
	-0.5, -0.5, -0.5, 0.0, 0.0, 0.0, 0.0, -1.0,
	0.5, -0.5, -0.5, 1, 0.0, 0.0, 0.0, -1.0,
	-0.5, -0.5, 0.5, 0.0, 1, 0.0, 0.0, -1.0,
	0.5, -0.5, -0.5, 1, 0.0, 0.0, 0.0, -1.0,
	0.5, -0.5, 0.5, 1.0, 1.0, 0.0, 0.0, -1.0,
	-0.5, -0.5, 0.5, 0.0, 1, 0.0, 0.0, -1.0,

	// Top
	-0.5, 0.5, -0.5, 0.0, 0.0, 0.0, 0.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1, 0.0, 0.0, 1.0,
	0.5, 0.5, -0.5, 1, 0.0, 0.0, 0.0, 1.0,
	0.5, 0.5, -0.5, 1, 0.0, 0.0, 0.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1, 0.0, 0.0, 1.0,
	0.5, 0.5, 0.5, 1, 1, 0.0, 0.0, 1.0,

	// Front
	-0.5, -0.5, 0.5, 1, 0.0, -1.0, 0.0, 0.0,
	0.5, -0.5, 0.5, 0.0, 0.0, -1.0, 0.0, 0.0,
	-0.5, 0.5, 0.5, 1, 1, -1.0, 0.0, 0.0,
	0.5, -0.5, 0.5, 0.0, 0.0, -1.0, 0.0, 0.0,
	0.5, 0.5, 0.5, 0.0, 1, -1.0, 0.0, 0.0,
	-0.5, 0.5, 0.5, 1, 1, -1.0, 0.0, 0.0,

	// Back
	-0.5, -0.5, -0.5, 0.0, 0.0, 1.0, 0.0, 0.0,
	-0.5, 0.5, -0.5, 0.0, 1, 1.0, 0.0, 0.0,
	0.5, -0.5, -0.5, 1, 0.0, 1.0, 0.0, 0.0,
	0.5, -0.5, -0.5, 1, 0.0, 1.0, 0.0, 0.0,
	-0.5, 0.5, -0.5, 0.0, 1, 1.0, 0.0, 0.0,
	0.5, 0.5, -0.5, 1, 1, 1.0, 0.0, 0.0,

	// Left
	-0.5, -0.5, 0.5, 0.0, 1, 0.0, -1.0, 0.0,
	-0.5, 0.5, -0.5, 1, 0.0, 0.0, -1.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0, 0.0, -1.0, 0.0,
	-0.5, -0.5, 0.5, 0.0, 1, 0.0, -1.0, 0.0,
	-0.5, 0.5, 0.5, 1, 1, 0.0, -1.0, 0.0,
	-0.5, 0.5, -0.5, 1, 0.0, 0.0, -1.0, 0.0,

	// Right
	0.5, -0.5, 0.5, 1, 1, 0.0, 1.0, 0.0,
	0.5, -0.5, -0.5, 1, 0.0, 0.0, 1.0, 0.0,
	0.5, 0.5, -0.5, 0.0, 0.0, 0.0, 1.0, 0.0,
	0.5, -0.5, 0.5, 1, 1, 0.0, 1.0, 0.0,
	0.5, 0.5, -0.5, 0.0, 0.0, 0.0, 1.0, 0.0,
	0.5, 0.5, 0.5, 0.0, 1, 0.0, 1.0, 0.0,
}
