package to.replicate.helloAndroidJava;

import androidx.appcompat.app.AppCompatActivity;

import android.app.AlertDialog;
import android.os.Bundle;
import android.util.Log;

import org.json.JSONTokener;

import java.io.File;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
    }

    @Override
    protected void onResume() {
        super.onResume();

        String message = "Error";
        try {
            File f = this.getFileStreamPath("db1");
            repm.Connection conn = repm.Repm.open(f.getAbsolutePath());
            repm.Command cmd = conn.exec("code/put", null);
            cmd.write("function setGreeting(greet) { db.put('greet', greet); }".getBytes());
            cmd.done();

            cmd = conn.exec("code/run", "{\"Name\": \"setGreeting\", \"Args\": [\"Aloha\"]}".getBytes());
            cmd.done();

            cmd = conn.exec("data/get", "{\"ID\": \"greet\"}".getBytes());
            byte[] buf = new byte[1024];
            long n = cmd.read(buf);
            message = (String)new JSONTokener(new String(buf, 0, (int)n)).nextValue() + ", Replicant!";
        } catch (Exception e) {
            Log.e("blech", e.toString());
        }

        new AlertDialog.Builder(this)
                .setTitle("beep")
                .setMessage(message)
                .show();
    }

}
