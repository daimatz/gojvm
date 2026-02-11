import java.util.ArrayList;

public class ObjectMethods {
    static class Point {
        int x, y;
        Point(int x, int y) {
            this.x = x;
            this.y = y;
        }
        public String toString() {
            return "(" + x + "," + y + ")";
        }
        public boolean equals(Object o) {
            if (!(o instanceof Point)) return false;
            Point p = (Point) o;
            return this.x == p.x && this.y == p.y;
        }
        public int hashCode() {
            return x * 31 + y;
        }
    }

    public static void main(String[] args) {
        Point p1 = new Point(3, 4);
        Point p2 = new Point(3, 4);
        Point p3 = new Point(1, 2);

        // toString
        System.out.println(p1);           // (3,4)
        System.out.println(p3);           // (1,2)

        // equals
        System.out.println(p1.equals(p2)); // true
        System.out.println(p1.equals(p3)); // false

        // hashCode
        System.out.println(p1.hashCode() == p2.hashCode()); // true

        // Use in ArrayList with indexOf
        ArrayList<Point> list = new ArrayList<>();
        list.add(p1);
        list.add(p3);
        System.out.println(list.size());   // 2
        System.out.println(list.get(0));   // (3,4)
        System.out.println(list.get(1));   // (1,2)
    }
}
