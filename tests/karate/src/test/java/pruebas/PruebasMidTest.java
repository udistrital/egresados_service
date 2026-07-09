package pruebas;

import com.intuit.karate.core.MockServer;
import com.intuit.karate.junit5.Karate;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;

/**
 * Runner de la suite Karate del MID.
 *
 * Antes de correr los features levanta el MOCK de los servicios institucionales
 * (WSO2 userinfo, autenticacion_mid/userRol, administrativa_amazon_api,
 * terceros_crud, sga_mid y gestor_documental_mid) en el puerto 8090. El MID
 * debe arrancarse con las variables de entorno apuntando a ese mock — el script
 * run_pruebas.ps1 lo hace automáticamente.
 *
 * Los features se listan de forma EXPLÍCITA para garantizar un orden
 * determinista (comparten la base de datos local re-sembrada con
 * db/seed_pruebas.sql) y se ejecutan en un solo hilo.
 */
class PruebasMidTest {

    static MockServer mock;

    @BeforeAll
    static void iniciarMockInstitucional() {
        mock = MockServer
                .feature("classpath:mocks/institucional-mock.feature")
                .http(8090)
                .build();
    }

    @AfterAll
    static void detenerMockInstitucional() {
        if (mock != null) {
            mock.stop();
        }
    }

    @Karate.Test
    Karate pruebas() {
        return Karate.run(
                "classpath:features/01-seguridad-autenticacion.feature",
                "classpath:features/02-catalogos.feature",
                "classpath:features/03-provision-jit.feature",
                "classpath:features/04-beneficios-empresa.feature",
                "classpath:features/05-solicitudes-flujo.feature",
                "classpath:features/06-solicitudes-cancelar-rechazar.feature",
                "classpath:features/07-documentos-solicitud.feature",
                "classpath:features/08-rn010-limite-activas.feature"
        ).tags("~@ignore");
    }
}
