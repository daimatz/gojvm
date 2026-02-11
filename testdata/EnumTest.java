public class EnumTest {
    enum Color { RED, GREEN, BLUE }

    public static void main(String[] args) {
        Color c = Color.GREEN;
        System.out.println(c.ordinal());   // 1
        System.out.println(c.name());      // GREEN
        Color[] colors = Color.values();
        System.out.println(colors.length); // 3
    }
}
