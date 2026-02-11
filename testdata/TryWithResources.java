public class TryWithResources {
    static class MyResource implements AutoCloseable {
        String name;
        MyResource(String name) {
            this.name = name;
            System.out.println("open " + name);
        }
        public void close() {
            System.out.println("close " + name);
        }
        void use() {
            System.out.println("use " + name);
        }
    }

    public static void main(String[] args) {
        try (MyResource r = new MyResource("A")) {
            r.use();
        }
        System.out.println("done");
    }
}
