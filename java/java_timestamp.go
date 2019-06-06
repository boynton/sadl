package java

func (gen *Generator) CreateTimestamp() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.Emit(javaTimestamp)
	result := gen.End()
	if gen.Err == nil {
		gen.WriteJavaFile("Timestamp", result, gen.Package)
	}
}

var javaTimestamp = `
import java.time.Instant;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.core.JsonProcessingException;
import java.io.IOException;

@JsonSerialize(using = Timestamp.Serializer.class)
@JsonDeserialize(using = Timestamp.Deserializer.class)
public class Timestamp implements Comparable<Timestamp> {
    private String repr;

    public static Timestamp now() {
        return new Timestamp(Instant.now());
    }

    public Timestamp(String repr) {
        this.repr = repr;
    }

    public Timestamp(Instant instant) {
        this.repr = instant.toString();
    }
    
    public String toString() {
        return repr;
    }

    public int compareTo(Timestamp another) {
        if (another == null) {
            return 1;
        }
        return this.asInstant().compareTo(another.asInstant());
    }

    public Instant asInstant() {
        return Instant.parse(repr);
    }

    public static class Serializer extends JsonSerializer<Timestamp> {
        @Override
        public void serialize(Timestamp value, JsonGenerator jgen, SerializerProvider provider) throws IOException, JsonProcessingException {
            jgen.writeString(value.repr);
        }
    }
    public static class Deserializer extends JsonDeserializer<Timestamp> {
        @Override
        public Timestamp deserialize(JsonParser jp, DeserializationContext ctxt) throws IOException, JsonProcessingException {
            String s = jp.getText();
            return new Timestamp(s);
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

}
`
