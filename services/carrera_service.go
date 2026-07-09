package services

import (
	"fmt"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// datosEstudianteResp es la forma cruda de academica_jbpm/datos_estudiante/{codigo}.
// Solo se declara el campo que se usa (Carrera); el resto del payload del SGA se
// descarta al deserializar.
type datosEstudianteResp struct {
	EstudianteCollection struct {
		DatosEstudiante []struct {
			Carrera string `json:"carrera"`
		} `json:"datosEstudiante"`
	} `json:"estudianteCollection"`
}

// carreraResp es la forma cruda de academica_jbpm/carrera/{codigo}.
type carreraResp struct {
	CarrerasCollection struct {
		Carrera []struct {
			Nombre string `json:"nombre"`
		} `json:"carrera"`
	} `json:"carrerasCollection"`
}

// conectoresMenores van en minúscula dentro del nombre de la carrera, salvo que
// abran la frase (primera palabra, o primera palabra tras un paréntesis).
var conectoresMenores = map[string]bool{
	"de": true, "del": true, "la": true, "las": true, "el": true, "los": true,
	"en": true, "y": true, "a": true, "al": true, "con": true, "para": true,
}

// ResolverCarrera encadena academica_jbpm (datos_estudiante → carrera) para obtener
// el nombre de la carrera de un egresado a partir de su código institucional.
// Best-effort para el caller: cualquier fallo o dato faltante devuelve error.
func ResolverCarrera(token, codigoInstitucional string) (string, error) {
	codigo := strings.TrimSpace(codigoInstitucional)
	if codigo == "" {
		return "", fmt.Errorf("código institucional vacío")
	}

	var estudiante datosEstudianteResp
	if err := helpers.GetAcademicaJbpm(token, "/datos_estudiante/"+codigo, &estudiante); err != nil {
		return "", fmt.Errorf("no se pudo consultar datos_estudiante de %s: %v", codigo, err)
	}
	datos := estudiante.EstudianteCollection.DatosEstudiante
	if len(datos) == 0 || strings.TrimSpace(datos[0].Carrera) == "" {
		return "", fmt.Errorf("datos_estudiante de %s no trajo carrera", codigo)
	}
	codigoCarrera := strings.TrimSpace(datos[0].Carrera)

	var carrera carreraResp
	if err := helpers.GetAcademicaJbpm(token, "/carrera/"+codigoCarrera, &carrera); err != nil {
		return "", fmt.Errorf("no se pudo consultar carrera %s: %v", codigoCarrera, err)
	}
	nombres := carrera.CarrerasCollection.Carrera
	if len(nombres) == 0 || strings.TrimSpace(nombres[0].Nombre) == "" {
		return "", fmt.Errorf("carrera %s no trajo nombre", codigoCarrera)
	}
	return formatearCarrera(nombres[0].Nombre), nil
}

// formatearCarrera pasa un nombre en MAYÚSCULAS (tal como lo entrega academica_jbpm)
// a Title Case en español: capitaliza cada palabra salvo los conectores menores
// (de, en, la, y, ...), que quedan en minúscula excepto si abren la frase o van
// justo después de un paréntesis. No restaura tildes: el dato fuente no las trae.
func formatearCarrera(s string) string {
	palabras := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	for i, p := range palabras {
		prefijo := ""
		nucleo := p
		for len(nucleo) > 0 && nucleo[0] == '(' {
			prefijo += "("
			nucleo = nucleo[1:]
		}
		sufijo := ""
		for len(nucleo) > 0 && (nucleo[len(nucleo)-1] == ')' || nucleo[len(nucleo)-1] == ',') {
			sufijo = string(nucleo[len(nucleo)-1]) + sufijo
			nucleo = nucleo[:len(nucleo)-1]
		}
		abreFrase := i == 0 || prefijo != ""
		if nucleo != "" && (abreFrase || !conectoresMenores[nucleo]) {
			nucleo = strings.ToUpper(nucleo[:1]) + nucleo[1:]
		}
		palabras[i] = prefijo + nucleo + sufijo
	}
	return strings.Join(palabras, " ")
}
