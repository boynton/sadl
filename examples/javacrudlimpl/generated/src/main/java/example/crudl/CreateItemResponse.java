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

@JsonDeserialize(builder = CreateItemResponse.CreateItemResponseBuilder.class)
public class CreateItemResponse {
    @NotNull
    private final Item item;

    public CreateItemResponse(Item item) {
        this.item = item;
    }

    public Item getItem() {
        return item;
    }

    public static CreateItemResponseBuilder builder() {
        return new CreateItemResponseBuilder();
    }

    @JsonPOJOBuilder(withPrefix="")
    public static class CreateItemResponseBuilder {
        private Item item;

        public CreateItemResponseBuilder item(Item item) {
            this.item = item;
            return this;
        }

        public CreateItemResponse build() {
            return new CreateItemResponse(item);
        }
    }
}