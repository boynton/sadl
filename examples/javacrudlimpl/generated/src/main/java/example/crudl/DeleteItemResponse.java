//
// Generated by sadl
//
package example.crudl;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonPOJOBuilder;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import javax.validation.constraints.NotNull;

@JsonDeserialize(builder = DeleteItemResponse.DeleteItemResponseBuilder.class)
public class DeleteItemResponse {
    public DeleteItemResponse() {
    }

    public static DeleteItemResponseBuilder builder() {
        return new DeleteItemResponseBuilder();
    }

    @JsonPOJOBuilder(withPrefix="")
    public static class DeleteItemResponseBuilder {

        public DeleteItemResponse build() {
            return new DeleteItemResponse();
        }
    }
}
