package java

func (gen *Generator) CreateJsonUtil() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.Emit(javaJsonUtil)
	result := gen.End()
	if gen.Err == nil {
		gen.WriteJavaFile("Json", result, gen.Package)
	}
}

var javaJsonUtil = `
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;

public class Json {

    static final ObjectMapper om = initMapper();
    static ObjectMapper initMapper() {
        ObjectMapper om = new ObjectMapper();
        om.disable(SerializationFeature.WRITE_NULL_MAP_VALUES);
        om.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        return om;
    }

    public static <T> T parse(String jsonData, Class<T> dataType) {
        try {
            return om.readerFor(dataType).readValue(jsonData);
        } catch (Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static String string(Object o) {
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

    public static <T> String[] validate(T t) {
        return new String[0]; //replace with a real validator
    }
}
`

