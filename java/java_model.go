package java

import (
	"text/template"

	"github.com/boynton/sadl"
)

func (gen *Generator) CreateModel() {
	var exceptions map[string]string
	if !gen.Config.GetBool("service-exception") {
		//generate all the pojo types that need to be exceptions
		exceptions = gen.ExceptionTypes()
	}
	for _, td := range gen.Model.Types {
		gen.CreatePojoFromDef(td, exceptions)
	}
	gen.CreateInterface()
	if gen.NeedTimestamp {
		gen.CreateTimestamp()
	} else if gen.NeedInstant {
		gen.NeedUtil = true
	}
	if gen.NeedUtil {
		gen.CreateUtil()
	}
	if gen.Config.GetBool("service-exception") {
		funcMap := template.FuncMap{}
		gen.CreateJavaFileFromTemplate("ServiceException", exceptionTemplate, gen, funcMap, gen.ModelPackage)
	}
}

func (gen *Generator) entityNameType(hact *sadl.HttpDef) (string, string) {
	for _, out := range hact.Expected.Outputs {
		if out.Header == "" {
			tn, _, _ := gen.TypeName(nil, out.Type, true)
			return out.Name, tn
		}
	}
	return "", "void"
}

func (gen *Generator) CreateInterface() {
	if gen.Err != nil {
		return
	}
	if gen.Name == "" {
		return
	}
	funcMap := template.FuncMap{
		"handlerSig": func(hact *sadl.HttpDef) string {
			name := gen.ActionName(hact)
			resType := gen.ResponseType(name)
			reqType := gen.RequestType(name)
			return "public " + resType + " " + name + "(" + reqType + " req)"
		},
	}
	gen.CreateJavaFileFromTemplate(gen.Name, interfaceTemplate, gen, funcMap, gen.ModelPackage)
	for _, hact := range gen.Model.Http {
		gen.CreateRequestPojo(hact)
		gen.CreateResponsePojo(hact)
		_, etype := gen.entityNameType(hact)
		if etype == "String" {
			gen.NeedUtil = true
		}
	}
}

const interfaceTemplate = `
public interface {{.Name}} {
{{range .Model.Http}}
    {{handlerSig .}};
{{end}}
}
`

const exceptionTemplate = `
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.core.JsonProcessingException;
import java.io.IOException;

@JsonSerialize(using = ServiceException.JSONSerializer.class)
@JsonDeserialize(using = ServiceException.JSONDeserializer.class)
public class ServiceException extends RuntimeException {
    public Object entity;

    public ServiceException() { super(); }
    public ServiceException(String message) { super(message); }
    public ServiceException(String message, Throwable cause) { super(message, cause); }
    public ServiceException(Throwable cause) { super(cause); }

    public ServiceException entity(Object entity) { this.entity = entity; return this; }

    public static class ErrorObject {
        public String error;
        public ErrorObject(String error) { this.error = error; }
    }

    public static class JSONSerializer extends JsonSerializer<ServiceException> {
        @Override
        public void serialize(ServiceException value, JsonGenerator jgen, SerializerProvider provider) throws IOException, JsonProcessingException {
            Object tmp = value.entity;
            if (tmp == null) {
                tmp = new ErrorObject(value.getMessage());
            }
            jgen.writeObject(tmp);
        }
    }

    public static class JSONDeserializer extends JsonDeserializer<ServiceException> {
        @Override
        public ServiceException deserialize(JsonParser jp, DeserializationContext ctxt) throws IOException, JsonProcessingException {
            ErrorObject tmp = jp.readValueAs(ErrorObject.class);
            return new ServiceException(tmp.error);
        }
    }

}
`
