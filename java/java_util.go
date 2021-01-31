package java

func (gen *Generator) CreateUtil() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.Emit(javaUtil)
	result := gen.End()
	if gen.Err == nil {
		gen.WriteJavaFile("Util", result, gen.ModelPackage)
	}
}

var javaUtil = `
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.SerializerProvider;
import javax.ws.rs.ext.ParamConverter;
import javax.ws.rs.ext.ParamConverterProvider;
import javax.ws.rs.ext.Provider;
import java.lang.annotation.Annotation;
import java.lang.reflect.Type;
import java.time.Instant;
import java.util.UUID;
import java.io.IOException;

public class Util {

    static final ObjectMapper om = initMapper();
    static ObjectMapper initMapper() {
        ObjectMapper om = new ObjectMapper();
        om.setDefaultPropertyInclusion(JsonInclude.Value.construct(Include.ALWAYS, Include.NON_NULL));
        om.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        return om;
    }

    public static <T> T fromJson(String jsonData, Class<T> dataType) {
        try {
            return om.readerFor(dataType).readValue(jsonData);
        } catch (Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static String toJson(Object o) {
        try {
            Class<?> cls = (o == null)? Object.class : o.getClass();
            return om.writerWithView(cls).writeValueAsString(o);
        } catch (Exception e) {
            e.printStackTrace();
            return "?";
        }
    }

    public static String pretty(Object o) {
        try {
            Class<?> cls = (o == null)? Object.class : o.getClass();
            return om.writerWithView(cls).with(SerializationFeature.INDENT_OUTPUT).writeValueAsString(o);
        } catch (Exception e) {
            e.printStackTrace();
            return "?";
        }
    }

    public static class InstantSerializer extends JsonSerializer<Instant> {
        @Override
        public void serialize(Instant value, JsonGenerator jgen, SerializerProvider provider) throws IOException, JsonProcessingException {
            jgen.writeString(value.toString());
        }
    }

    public static class InstantDeserializer extends JsonDeserializer<Instant> {
        @Override
        public Instant deserialize(JsonParser jp, DeserializationContext ctxt) throws IOException, JsonProcessingException {
            String s = jp.getText();
            return Instant.parse(s);
        }
    }

    @Provider
    public static class InstantConverterProvider implements ParamConverterProvider {
        @Override
        @SuppressWarnings("unchecked")
        public <T> ParamConverter<T> getConverter(Class<T> rawType, Type genericType, Annotation[] annotations) {
            if (rawType.equals(Instant.class))
                return (ParamConverter<T>) new InstantConverter();
            return null;
        }
    }

    public static class InstantConverter implements ParamConverter<Instant> {
        @Override
        public Instant fromString(String value) {
            if (value != null) {
                return Instant.parse(value);
            }
            return null;
        }
        @Override
        public String toString(Instant value) {
            return value.toString();
        }
    }

    public static <T> String[] validate(T t) {
        return new String[0]; //TO DO
    }
}
`
