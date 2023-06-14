import com.linecorp.armeria.common.HttpRequest;
import com.linecorp.armeria.common.HttpResponse;
import com.linecorp.armeria.common.HttpStatus;
import com.linecorp.armeria.common.MediaType;

import com.linecorp.armeria.server.AbstractHttpService;
import com.linecorp.armeria.server.Server;
import com.linecorp.armeria.server.ServerBuilder;
import com.linecorp.armeria.server.ServiceRequestContext;

import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.sql.*;
import java.util.Map;
import java.util.AbstractMap;
import java.util.NoSuchElementException;
import java.util.concurrent.CompletableFuture;

public class Main {

    private static Map<String, String> tables = Map.ofEntries(
        new AbstractMap.SimpleEntry<String, String>("noun", "nouns"),
        new AbstractMap.SimpleEntry<String, String>("verb", "verbs"),
        new AbstractMap.SimpleEntry<String, String>("adjective", "adjectives")
    );
    public static void main(String[] args) throws Exception {
        Class.forName("org.postgresql.Driver");

        ServerBuilder sb = Server.builder();
        // Configure an HTTP port.
        sb.http(8080);
        sb.service("/{wordtype}", new AbstractHttpService() {
            @Override
            protected HttpResponse doGet(ServiceRequestContext ctx, HttpRequest req) {
                String wordType = ctx.pathParam("wordtype");
                String word = randomWord(wordType);
                return HttpResponse.of(HttpStatus.OK, MediaType.JSON_UTF_8,
                               "{\"word\":\"%s\"}", word);
            }
        });
        Server server = sb.build();
        CompletableFuture<Void> future = server.start();
        future.join();
    }

    private static String randomWord(String wordType) {
        String table = tables.get(wordType);
        try (Connection connection = DriverManager.getConnection("jdbc:postgresql://db:5432/postgres", "postgres", "")) {
            try (Statement statement = connection.createStatement()) {
                try (ResultSet set = statement.executeQuery("SELECT word FROM " + table + " ORDER BY random() LIMIT 1")) {
                    while (set.next()) {
                        return set.getString(1);
                    }
                }
            }
        } catch (SQLException e) {
            e.printStackTrace();
        }

        throw new NoSuchElementException(table);
    }

}
